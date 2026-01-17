package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/roveo/wt/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tracked worktrees",
	Long:    `List all worktrees tracked by wt across all repositories.`,
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	database, err := db.Default()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	worktrees, err := db.ListAllWorktrees(database)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees tracked. Run 'wt' inside a git repository to index it.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPO\tBRANCH\tPATH")
	for _, wt := range worktrees {
		branch := wt.Branch
		if wt.IsMain {
			branch += " [main]"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", wt.RepoName, branch, wt.Path)
	}
	w.Flush()

	return nil
}
