package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"git-autometa/internal/github"
	"git-autometa/internal/jira"
)

var (
	baseBranch string
	noDraft    bool
)

var createPrCmd = &cobra.Command{
	Use:          "create-pr",
	Short:        "Create a pull request for the current branch",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runCreatePR(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	createPrCmd.Flags().StringVar(&baseBranch, "base-branch", "", "Base branch for the PR (overrides config)")
	createPrCmd.Flags().BoolVar(&noDraft, "no-draft", false, "Create a non-draft PR")
	rootCmd.AddCommand(createPrCmd)
}

func runCreatePR(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	_ = cfg // placeholder until fully implemented

	jiraClient, err := jira.NewClientWithKeyring(cfg)
	if err != nil {
		return err
	}
	ghClient := github.NewClient(cfg)

	_ = jiraClient
	_ = ghClient
	_ = baseBranch
	_ = noDraft

	fmt.Println("create-pr scaffolding executed. Implementation pending.")
	return nil
}
