package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/git"
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

	jiraClient, err := jira.NewClientWithKeyring(cfg)
	if err != nil {
		return err
	}
	ghClient := github.NewClient(cfg)
	gitUtils := git.New()

	url, err := createPRWithDeps(cfg, jiraClient, gitUtils, ghClient, baseBranch, noDraft)
	if err != nil {
		return err
	}
	fmt.Println("Created PR:", url)
	return nil
}

// --- dependency injected core for testing ---

type jiraGetter interface {
	GetIssue(key string) (*jira.Issue, error)
}

// gitContext unifies git information and commit listing needs.
type gitContext interface {
	GetCurrentBranch() (string, error)
	GetCommitMessagesForPR(baseBranch string) ([]string, error)
}

type ghCreator interface {
	CreatePullRequest(title, body, head, base string, draft bool) (string, error)
}

func createPRWithDeps(
	cfg appconfig.Config,
	jc jiraGetter,
	gu gitContext,
	gh ghCreator,
	overrideBase string,
	forceNoDraft bool,
) (string, error) {
	base := overrideBase
	if base == "" {
		base = cfg.PullRequest.BaseBranch
	}
	draft := cfg.PullRequest.Draft
	if forceNoDraft {
		draft = false
	}

	headBranch, err := gu.GetCurrentBranch()
	if err != nil {
		return "", err
	}
	issueKey, ok := extractIssueKeyFromBranch(headBranch)
	if !ok {
		return "", fmt.Errorf("unable to determine JIRA key from branch %q", headBranch)
	}
	issue, err := jc.GetIssue(issueKey)
	if err != nil {
		return "", err
	}
	title := formatPRTitle(cfg, *issue)
	body, err := formatPRBody(cfg, gu, base, *issue)
	if err != nil {
		return "", err
	}
	return gh.CreatePullRequest(title, body, headBranch, base, draft)
}

// extractIssueKeyFromBranch finds the first occurrence of an uppercase JIRA key like ABC-123.
func extractIssueKeyFromBranch(branch string) (string, bool) {
	re := regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)
	m := re.FindString(branch)
	if m == "" {
		return "", false
	}
	return m, true
}

func formatPRTitle(cfg appconfig.Config, issue jira.Issue) string {
	pattern := cfg.PullRequest.TitlePattern
	titleSlug := issue.SlugifyTitle(0)
	out := pattern
	out = strings.ReplaceAll(out, "{jira_id}", issue.Key)
	out = strings.ReplaceAll(out, "{jira_title}", titleSlug)
	out = strings.ReplaceAll(out, "{jira_type}", strings.ToLower(strings.TrimSpace(issue.IssueType)))
	out = strings.TrimSpace(out)
	if out == "" {
		if titleSlug != "" {
			return fmt.Sprintf("%s: %s", issue.Key, titleSlug)
		}
		return issue.Key
	}
	return out
}

func formatPRBody(cfg appconfig.Config, cl gitContext, base string, issue jira.Issue) (string, error) {
	template := cfg.PullRequest.Template
	if template == "" {
		return "", errors.New("empty PR template in configuration")
	}
	commits, err := cl.GetCommitMessagesForPR(base)
	if err != nil {
		return "", err
	}
	var commitSection string
	for i, msg := range commits {
		if i == 0 {
			commitSection = "- " + msg
		} else {
			commitSection += "\n- " + msg
		}
	}
	desc := issue.DescriptionMarkdown()
	out := template
	out = strings.ReplaceAll(out, "{jira_id}", issue.Key)
	out = strings.ReplaceAll(out, "{jira_title}", issue.SlugifyTitle(0))
	out = strings.ReplaceAll(out, "{jira_type}", strings.ToLower(strings.TrimSpace(issue.IssueType)))
	out = strings.ReplaceAll(out, "{jira_url}", strings.TrimSpace(issue.URL))
	out = strings.ReplaceAll(out, "{jira_description}", desc)
	out = strings.ReplaceAll(out, "{commit_messages}", commitSection)
	return strings.TrimSpace(out), nil
}
