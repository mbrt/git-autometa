package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/github"
	"git-autometa/internal/jira"
)

var (
	baseBranch string
	noDraft    bool
)

var createPrCmd = &cobra.Command{
	Use:   "create-pr",
	Short: "Create a pull request for the current branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.LoadEffectiveConfig(cfgPath)
		if err != nil {
			return err
		}

		_ = cfg // placeholder

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
	},
}

func init() {
	createPrCmd.Flags().StringVar(&baseBranch, "base-branch", "", "Base branch for the PR (overrides config)")
	createPrCmd.Flags().BoolVar(&noDraft, "no-draft", false, "Create a non-draft PR")
	rootCmd.AddCommand(createPrCmd)
}
