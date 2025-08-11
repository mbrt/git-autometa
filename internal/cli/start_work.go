package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/gitutils"
	"git-autometa/internal/jira"
)

var (
	pushFlag bool
)

var startWorkCmd = &cobra.Command{
	Use:   "start-work [JIRA-KEY]",
	Short: "Start work on a JIRA issue by creating and switching to a branch",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := appconfig.LoadEffectiveConfig(cfgPath)
		if err != nil {
			return err
		}

		_ = cfg // placeholder to avoid unused for now

		jiraClient, err := jira.NewClientWithKeyring(cfg)
		if err != nil {
			return err
		}
		git := gitutils.NewUtils()

		// Placeholder flow mirroring architecture (no real behavior yet)
		var issueKey string
		if len(args) == 1 {
			issueKey = args[0]
		} else {
			fmt.Println("Interactive selection is not implemented yet; pass a JIRA key.")
			return nil
		}

		_ = jiraClient
		_ = git
		_ = issueKey

		fmt.Println("start-work scaffolding executed. Implementation pending.")
		return nil
	},
}

func init() {
	startWorkCmd.Flags().BoolVar(&pushFlag, "push", false, "Push the branch after creation")
	rootCmd.AddCommand(startWorkCmd)
}
