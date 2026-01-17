package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/roveo/wt/internal/db"
	"github.com/roveo/wt/internal/git"
	"github.com/roveo/wt/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [branch]",
	Short: "Add a new worktree",
	Long: `Add a new worktree for the current repository.

If branch is not specified, an interactive picker will be shown
to select from available remote branches, or you can enter a new branch name.

The worktree will be created at ../{repo}.worktrees/{branch}`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a git repo
	if !git.IsInsideRepo(cwd) {
		return fmt.Errorf("not inside a git repository")
	}

	// Get main repo path
	mainRepoPath, err := git.GetMainRepoPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to get main repo path: %w", err)
	}

	// If branch provided as argument, use it directly
	if len(args) > 0 {
		return runAddWithBranchFromRepo(mainRepoPath, args[0])
	}

	// Otherwise, run interactive workflow using a synthetic worktree for current repo
	database, err := db.Default()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get current repo from database
	repo, err := db.GetRepoByPath(database, mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to get repo: %w", err)
	}
	if repo == nil {
		// Ensure repo is in DB first
		if err := ensureCurrentRepoInDB(database, cwd); err != nil {
			return err
		}
		repo, _ = db.GetRepoByPath(database, mainRepoPath)
	}

	// Create a worktree reference for the add workflow
	sourceWorktree := &db.Worktree{
		RepoPath: mainRepoPath,
		RepoName: repo.Name,
	}

	_, err = runAddWorkflow(sourceWorktree)
	return err
}

// runAddWorkflow runs the interactive add workflow
// sourceWorktree is the worktree/repo to create the new worktree from (can be nil for CLI usage)
// Returns ui.ActionBack if user wants to go back to worktree list
func runAddWorkflow(sourceWorktree *db.Worktree) (ui.PickerAction, error) {
	if sourceWorktree == nil {
		return ui.ActionNone, fmt.Errorf("no source worktree selected")
	}

	// Show interactive picker for branch name
	branch, action, err := ui.InputBranch("feature/my-branch", sourceWorktree.RepoName, sourceWorktree.Branch)
	if err != nil {
		return ui.ActionNone, err
	}
	if action == ui.ActionBack {
		return ui.ActionBack, nil
	}
	if branch == "" {
		// User cancelled
		return ui.ActionNone, nil
	}

	return ui.ActionNone, runAddWithBranchFromRepo(sourceWorktree.RepoPath, branch)
}

// runAddWithBranchFromRepo creates a worktree for the given branch from the specified repo
func runAddWithBranchFromRepo(repoPath, branch string) error {
	// Open database and ensure repo is indexed
	database, err := db.Default()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := ensureCurrentRepoInDB(database, repoPath); err != nil {
		return err
	}

	// Sanitize branch name for directory (replace / with -)
	dirName := strings.ReplaceAll(branch, "/", "-")

	// Determine target path
	worktreesDir := git.GetDefaultWorktreesDir(repoPath)
	targetPath := filepath.Join(worktreesDir, dirName)

	// Check if target already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", targetPath)
	}

	// Ensure worktrees directory exists
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Create worktree
	fmt.Fprintf(os.Stderr, "Creating worktree for branch '%s' at %s...\n", branch, targetPath)
	if err := git.AddWorktree(repoPath, branch, targetPath); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Sync to update database
	repo, err := db.GetRepoByPath(database, repoPath)
	if err != nil || repo == nil {
		return fmt.Errorf("failed to get repo from database: %w", err)
	}
	if err := syncWorktrees(database, repo); err != nil {
		return fmt.Errorf("failed to sync worktrees: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Worktree created successfully.\n")

	// Output cd command
	fmt.Printf("cd %q\n", targetPath)
	return nil
}
