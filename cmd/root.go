package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/roveo/wt/internal/config"
	"github.com/roveo/wt/internal/db"
	"github.com/roveo/wt/internal/git"
	"github.com/roveo/wt/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wt",
	Short: "Lightweight Git worktree manager",
	Long: `wt is a lightweight and agile tool to manage Git worktrees.

Run 'wt' without arguments to:
  - Auto-index the current repository
  - Show a fuzzy finder to switch between worktrees
  - Output a cd command for shell integration

Setup shell integration by adding to your rc file:
  eval "$(wt init bash)"   # or zsh/fish`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Open database
	database, err := db.Default()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// === SYNC PHASE ===
	// 1. If inside a repo, ensure it's in the database (use main repo path)
	if git.IsInsideRepo(cwd) {
		if err := ensureCurrentRepoInDB(database, cwd); err != nil {
			return err
		}
	}

	// 2. Sync all repos (add/soft-delete worktrees)
	if err := syncAllRepos(database); err != nil {
		return err
	}

	// === DISPLAY PHASE (uses only SQLite data) ===
	// Get current repo path for sorting (current repo's worktrees first)
	var currentRepoPath string
	if git.IsInsideRepo(cwd) {
		currentRepoPath, _ = git.GetMainRepoPath(cwd)
	}

	// Get all worktrees from database (current repo first)
	worktrees, err := db.ListAllWorktreesWithRepoFirst(database, currentRepoPath)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// If no worktrees found, go directly to add workflow if we're in a repo
	if len(worktrees) == 0 {
		if !git.IsInsideRepo(cwd) {
			return fmt.Errorf("no worktrees found - run 'wt' inside a git repository to index it")
		}
		// Go directly to add workflow
		_, err := runAddWorkflow(nil)
		return err
	}

	// Main loop - allows switching between worktree picker and add mode
	for {
		// Show picker
		result, err := ui.PickWorktree(worktrees)
		if err != nil {
			return err
		}

		switch result.Action {
		case ui.ActionNone:
			// User cancelled
			return nil
		case ui.ActionSwitch:
			if result.Worktree == nil {
				return nil
			}
			outputWorktreeSwitch(result.Worktree.Path, result.Worktree.RepoPath)
			return nil
		case ui.ActionAdd:
			// Switch to add workflow - create worktree from the selected repo
			if result.Worktree == nil {
				continue
			}
			action, err := runAddWorkflow(result.Worktree)
			if err != nil {
				return err
			}
			// If user pressed back, continue the loop to show worktree picker again
			if action == ui.ActionBack {
				continue
			}
			// Otherwise (user completed add or cancelled), exit
			return nil
		case ui.ActionDelete:
			// Delete the selected worktree
			if result.Worktree == nil {
				continue
			}
			if err := deleteWorktree(database, result.Worktree); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			// Refresh worktree list and continue
			worktrees, err = db.ListAllWorktreesWithRepoFirst(database, currentRepoPath)
			if err != nil {
				return fmt.Errorf("failed to list worktrees: %w", err)
			}
			if len(worktrees) == 0 {
				return nil
			}
			continue
		}
	}
}

// ensureCurrentRepoInDB ensures the current repository is added to the database
// If inside a worktree, it finds and adds the main repository
func ensureCurrentRepoInDB(database *sql.DB, cwd string) error {
	// Get main repo path (works from both main repo and worktrees)
	mainRepoPath, err := git.GetMainRepoPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to get main repo path: %w", err)
	}

	// Check if repo exists in database
	repo, err := db.GetRepoByPath(database, mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to check repo: %w", err)
	}

	// If not in DB, add it
	if repo == nil {
		repo = &db.Repo{
			Path:         mainRepoPath,
			Name:         git.GetRepoName(mainRepoPath),
			WorktreesDir: git.GetDefaultWorktreesDir(mainRepoPath),
		}
		if err := db.UpsertRepo(database, repo); err != nil {
			return fmt.Errorf("failed to save repo: %w", err)
		}
	}

	return nil
}

// syncAllRepos syncs worktrees for all repositories in the database
func syncAllRepos(database *sql.DB) error {
	repos, err := db.ListRepos(database)
	if err != nil {
		return fmt.Errorf("failed to list repos: %w", err)
	}

	for _, repo := range repos {
		if err := syncWorktrees(database, repo); err != nil {
			// Log error but continue with other repos
			fmt.Fprintf(os.Stderr, "warning: failed to sync %s: %v\n", repo.Name, err)
			continue
		}
		if err := db.UpdateLastSynced(database, repo.ID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to update sync timestamp for %s: %v\n", repo.Name, err)
		}
	}

	return nil
}

// deleteWorktree deletes a worktree with confirmation
func deleteWorktree(database *sql.DB, wt *db.Worktree) error {
	if wt.IsMain {
		return fmt.Errorf("cannot delete the main worktree")
	}

	// Confirm deletion
	confirmed, err := ui.Confirm(fmt.Sprintf("Delete worktree %s/%s?", wt.RepoName, wt.Branch))
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	// Remove from git
	fmt.Fprintf(os.Stderr, "Removing worktree...\n")
	if err := git.RemoveWorktree(wt.RepoPath, wt.Path); err != nil {
		// Try force remove
		if err := git.RemoveWorktreeForce(wt.RepoPath, wt.Path); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}

	// Soft-delete from database
	if err := db.SoftDeleteWorktree(database, wt.ID); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Worktree deleted.\n")
	return nil
}

// syncWorktrees syncs the worktrees for a repository
func syncWorktrees(database *sql.DB, repo *db.Repo) error {
	// Get worktrees from git
	gitWorktrees, err := git.ListWorktrees(repo.Path)
	if err != nil {
		return err
	}

	// Upsert each worktree
	var existingPaths []string
	for _, gwt := range gitWorktrees {
		wt := &db.Worktree{
			RepoID: repo.ID,
			Path:   gwt.Path,
			Branch: gwt.Branch,
			IsMain: gwt.IsMain,
		}
		if err := db.UpsertWorktree(database, wt); err != nil {
			return err
		}
		existingPaths = append(existingPaths, gwt.Path)
	}

	// Soft-delete worktrees that no longer exist
	if err := db.SoftDeleteMissingWorktrees(database, repo.ID, existingPaths); err != nil {
		return err
	}

	return nil
}

// outputWorktreeSwitch outputs the cd command and on_enter command for switching to a worktree
func outputWorktreeSwitch(worktreePath, repoPath string) {
	fmt.Printf("cd %q\n", worktreePath)
	projectCfg, _ := config.LoadProject(repoPath)
	if projectCfg.OnEnter != "" {
		fmt.Println(projectCfg.OnEnter)
	}
}
