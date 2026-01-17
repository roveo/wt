package cmd

import (
	"fmt"
	"os"

	"github.com/roveo/wt/internal/db"
	"github.com/roveo/wt/internal/git"
	"github.com/roveo/wt/internal/ui"
	"github.com/spf13/cobra"
)

var (
	removeForce bool
)

var removeCmd = &cobra.Command{
	Use:     "remove [worktree-path]",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree",
	Long: `Remove a worktree from the filesystem and database.

If no path is specified, an interactive picker will be shown.
The main worktree cannot be removed.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal even with uncommitted changes")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	database, err := db.Default()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get current directory to sync repo if we're in one
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Sync phase: ensure current repo is in DB, then sync all repos
	if git.IsInsideRepo(cwd) {
		if err := ensureCurrentRepoInDB(database, cwd); err != nil {
			return err
		}
	}
	if err := syncAllRepos(database); err != nil {
		return err
	}

	var worktree *db.Worktree

	if len(args) > 0 {
		// Path provided
		worktree, err = db.GetWorktreeByPath(database, args[0])
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		if worktree == nil {
			return fmt.Errorf("worktree not found: %s", args[0])
		}
	} else {
		// Interactive picker
		worktrees, err := db.ListAllWorktrees(database)
		if err != nil {
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		// Filter out main worktrees
		var removable []*db.Worktree
		for _, wt := range worktrees {
			if !wt.IsMain {
				removable = append(removable, wt)
			}
		}

		if len(removable) == 0 {
			return fmt.Errorf("no removable worktrees found (main worktrees cannot be removed)")
		}

		worktree, err = ui.PickWorktreeSimple(removable)
		if err != nil {
			return err
		}
		if worktree == nil {
			// User cancelled
			return nil
		}
	}

	// Check if it's the main worktree
	if worktree.IsMain {
		return fmt.Errorf("cannot remove the main worktree")
	}

	// Confirm removal
	confirmed, err := ui.Confirm(fmt.Sprintf("Remove worktree '%s/%s' at %s?", worktree.RepoName, worktree.Branch, worktree.Path))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove worktree from git
	fmt.Fprintf(os.Stderr, "Removing worktree...\n")
	var removeErr error
	if removeForce {
		removeErr = git.RemoveWorktreeForce(worktree.RepoPath, worktree.Path)
	} else {
		removeErr = git.RemoveWorktree(worktree.RepoPath, worktree.Path)
	}

	if removeErr != nil {
		return fmt.Errorf("failed to remove worktree: %w (use --force to force removal)", removeErr)
	}

	// Soft-delete from database
	if err := db.SoftDeleteWorktree(database, worktree.ID); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Worktree removed successfully.\n")
	return nil
}
