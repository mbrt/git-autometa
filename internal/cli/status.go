package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/git"
	"git-autometa/internal/github"
	"git-autometa/internal/jira"
	"git-autometa/internal/secrets"
)

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show repository and configuration status",
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStatus(cmd, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, _ []string) error {
	// Load effective configuration (global + repo overrides if available)
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Repository info
	fmt.Println("Repository:")
	gitUtils := git.New()
	if branch, err := gitUtils.GetCurrentBranch(); err == nil {
		fmt.Printf("  Current branch: %s\n", branch)
	} else {
		fmt.Println("  Current branch: (not a git repository)")
	}
	if remoteURL, err := gitUtils.GetRemoteURL("origin"); err == nil && remoteURL != "" {
		fmt.Printf("  Remote 'origin': %s\n", remoteURL)
	} else {
		fmt.Println("  Remote 'origin': (not set)")
	}

	owner, repo := resolveOwnerRepo()
	if owner != "" && repo != "" {
		fmt.Printf("  GitHub repository: %s/%s\n", owner, repo)
	} else {
		fmt.Println("  GitHub repository: (not detected)")
	}

	// Configuration paths
	fmt.Println()
	fmt.Println("Configuration:")
	globalPath := appconfig.GlobalConfigPath()
	if _, err := os.Stat(globalPath); err == nil {
		fmt.Printf("  Global config: %s\n", globalPath)
	} else {
		fmt.Printf("  Global config: %s (missing)\n", globalPath)
	}
	if owner != "" && repo != "" {
		repoPath := appconfig.RepoConfigPath(owner, repo)
		if _, err := os.Stat(repoPath); err == nil {
			fmt.Printf("  Repo config:   %s\n", repoPath)
		} else {
			fmt.Printf("  Repo config:   %s (missing)\n", repoPath)
		}
	}

	// Effective config summary (reuse existing config show functionality)
	if err := runConfigShowEffective(cmd.OutOrStdout()); err != nil {
		return err
	}

	// Credentials / Auth
	fmt.Println()
	fmt.Println("Credentials:")
	// Jira token presence (do not print token)
	jiraTokenStatus := "not configured"
	if cfg.Jira.Email != "" {
		if _, err := secrets.GetJiraToken(cfg.Jira.Email); err == nil {
			jiraTokenStatus = "present"
		} else {
			jiraTokenStatus = "missing"
		}
	}
	fmt.Printf("  Jira token: %s\n", jiraTokenStatus)

	// GitHub CLI authentication
	ghClient := github.NewClient(cfg)
	if err := ghClient.TestConnection(); err == nil {
		fmt.Println("  GitHub CLI auth: OK")
	} else {
		fmt.Printf("  GitHub CLI auth: ERROR: %v\n", err)
	}

	// Optional connectivity checks in verbose mode
	if verbose {
		fmt.Println()
		fmt.Println("Connectivity checks (verbose):")
		if cfg.Jira.Email != "" {
			jc, err := jira.NewClientWithKeyring(cfg)
			if err == nil {
				if err := jc.TestConnection(); err == nil {
					fmt.Println("  Jira API: OK")
				} else {
					fmt.Printf("  Jira API: ERROR: %v\n", err)
				}
			} else {
				fmt.Printf("  Jira API: ERROR: %v\n", err)
			}
		} else {
			fmt.Println("  Jira API: (email not configured)")
		}
	}

	return nil
}
