package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/git"
	"git-autometa/internal/jira"
)

var (
	pushFlag bool
)

var startWorkCmd = &cobra.Command{
	Use:          "start-work [JIRA-KEY]",
	Short:        "Start work on a JIRA issue by creating and switching to a branch",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runStartWork(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func init() {
	startWorkCmd.Flags().BoolVar(&pushFlag, "push", false, "Push the branch after creation")
	rootCmd.AddCommand(startWorkCmd)
}

// selectIssueInteractively lists assigned issues and lets the user pick one.
// Returns nil if user cancels.
// Narrow Jira interface for testability
type jiraService interface {
	SearchMyIssues(limit int) ([]jira.Issue, error)
	GetIssue(key string) (*jira.Issue, error)
}

// Narrow Git interface for testability
type gitService interface {
	PrepareWorkBranch(desiredBranchName string) (string, error)
	PushBranch(branchName string) error
}

func selectIssueInteractively(jc jiraService, in io.Reader, out io.Writer) (*jira.Issue, error) {
	issues, err := jc.SearchMyIssues(15)
	if err != nil || len(issues) == 0 {
		// Fallback: manual entry
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to fetch assigned issues: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "No assigned issues found.")
		}
		fmt.Fprint(out, "Enter a JIRA issue key (or leave empty to cancel): ")
		reader := bufio.NewReader(in)
		key := readString(reader)
		if key == "" {
			return nil, nil
		}
		issue, getErr := jc.GetIssue(key)
		if getErr != nil {
			return nil, getErr
		}
		return issue, nil
	}

	// Show list
	fmt.Fprintln(out, "Found assigned issues:")
	for idx, it := range issues {
		fmt.Fprintf(out, " %2d. %s: %s\n", idx+1, it.Key, truncateString(it.Summary, 90))
		if it.Status != "" || it.IssueType != "" {
			fmt.Fprintf(out, "     Status: %s  Type: %s\n", it.Status, it.IssueType)
		}
	}
	fmt.Fprintln(out, "  0. Cancel")

	// Prompt
	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(out, "Select an issue: ")
		choiceStr := readString(reader)
		if choiceStr == "" {
			continue
		}
		choice, convErr := strconv.Atoi(choiceStr)
		if convErr != nil || choice < 0 || choice > len(issues) {
			fmt.Fprintf(out, "Enter a number between 0 and %d\n", len(issues))
			continue
		}
		if choice == 0 {
			return nil, nil
		}
		// Convert to zero-based index
		selected := issues[choice-1]
		// Fetch full issue to get description and canonical URL if needed
		return jc.GetIssue(selected.Key)
	}
}

func formatBranchName(cfg appconfig.Config, issue jira.Issue) string {
	pattern := cfg.Git.BranchPattern
	maxLen := cfg.Git.MaxBranchLength
	titleSlug := issue.SlugifyTitle(maxLen)
	branch := strings.ReplaceAll(pattern, "{jira_id}", issue.Key)
	branch = strings.ReplaceAll(branch, "{jira_title}", titleSlug)
	branch = strings.ReplaceAll(branch, "{jira_type}", strings.ToLower(issue.IssueType))
	branch = sanitizeBranchName(branch)
	if maxLen > 0 && len(branch) > maxLen {
		branch = branch[:maxLen]
	}
	branch = strings.Trim(branch, "-")
	return branch
}

func runStartWork(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	jiraClient, err := jira.NewClientWithKeyring(cfg)
	if err != nil {
		return err
	}
	gitUtils := git.Git{MainBranch: cfg.PullRequest.BaseBranch}
	return startWorkWithDeps(args, cfg, jiraClient, gitUtils, os.Stdin, os.Stdout, pushFlag)
}

// startWorkWithDeps contains the testable core logic.
func startWorkWithDeps(args []string, cfg appconfig.Config, jc jiraService, gu gitService, in io.Reader, out io.Writer, push bool) error {
	// Resolve the issue either from CLI arg or interactively
	var (
		selectedIssue *jira.Issue
		err           error
	)
	if len(args) == 1 {
		issueKey := args[0]
		if issueKey == "" {
			return errors.New("empty JIRA key provided")
		}
		selectedIssue, err = jc.GetIssue(issueKey)
		if err != nil {
			return err
		}
	} else {
		selectedIssue, err = selectIssueInteractively(jc, in, out)
		if err != nil {
			return err
		}
		if selectedIssue == nil {
			// user cancelled
			return nil
		}
	}

	// Format desired branch name from config and issue
	desiredBranchName := formatBranchName(cfg, *selectedIssue)

	// Prepare and switch to work branch (auto-increments if exists)
	finalBranchName, err := gu.PrepareWorkBranch(desiredBranchName)
	if err != nil {
		return err
	}

	// Optionally push
	if push {
		if err := gu.PushBranch(finalBranchName); err != nil {
			return err
		}
	}

	// Output
	fmt.Fprintf(out, "Ready on branch: %s\n", finalBranchName)
	fmt.Fprintf(out, "Issue: %s - %s\n", selectedIssue.Key, selectedIssue.Summary)
	if selectedIssue.URL != "" {
		fmt.Fprintf(out, "Link: %s\n", selectedIssue.URL)
	}
	return nil
}

// sanitizeBranchName performs minimal cleanup to ensure a safe git branch name.
func sanitizeBranchName(s string) string {
	// Replace any non-alphanumeric character with a dash using a whitelist approach
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	return strings.Trim(b.String(), "-")
}

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
