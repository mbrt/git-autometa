package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose   bool
	repoOwner string
	repoName  string
)

var rootCmd = &cobra.Command{
	Use:   "git-autometa",
	Short: "Automate Git workflows around JIRA and GitHub",
	Long:  "git-autometa is a CLI to streamline branch creation and PR creation with JIRA and GitHub.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&repoOwner, "owner", "", "Repository owner (defaults to current git remote)")
	rootCmd.PersistentFlags().StringVar(&repoName, "repo", "", "Repository name (defaults to current git remote)")
}
