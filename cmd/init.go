package cmd

import (
	"fmt"

	"github.com/roveo/wt/internal/shell"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <shell>",
	Short: "Print shell initialization script",
	Long: `Print the shell initialization script for wt.

Add this to your shell's rc file:
  bash: eval "$(wt init bash)"   # add to ~/.bashrc
  zsh:  eval "$(wt init zsh)"    # add to ~/.zshrc
  fish: wt init fish | source    # add to ~/.config/fish/config.fish

This creates a shell wrapper function that allows wt to change
the current directory when switching worktrees.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		script, err := shell.GetInit(args[0])
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
