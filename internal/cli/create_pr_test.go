package cli

import (
	"strings"
	"testing"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/jira"
)

// --- fakes ---

type fakeJiraGetter struct {
	issue *jira.Issue
	err   error
}

func (f *fakeJiraGetter) GetIssue(key string) (*jira.Issue, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.issue != nil {
		return f.issue, nil
	}
	return &jira.Issue{Key: key, Summary: "Default", IssueType: "Task"}, nil
}

type fakeGitInfo struct {
	branch  string
	commits []string
	err     error
}

func (f *fakeGitInfo) GetCurrentBranch() (string, error)                    { return f.branch, f.err }
func (f *fakeGitInfo) GetCommitMessagesForPR(base string) ([]string, error) { return f.commits, nil }

type fakeGH struct {
	got struct {
		title, body, head, base string
		draft                   bool
	}
	url string
	err error
}

func (f *fakeGH) CreatePullRequest(title, body, head, base string, draft bool) (string, error) {
	f.got.title, f.got.body, f.got.head, f.got.base, f.got.draft = title, body, head, base, draft
	if f.err != nil {
		return "", f.err
	}
	if f.url == "" {
		f.url = "https://example/pr/1"
	}
	return f.url, nil
}

// --- tests ---

func TestExtractIssueKeyFromBranch(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"feature/APP-123-something", "APP-123", true},
		{"hotfix/XYZ1-9", "XYZ1-9", true},
		{"no-key-here", "", false},
	}
	for _, tc := range cases {
		got, ok := extractIssueKeyFromBranch(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("extractIssueKeyFromBranch(%q) = (%q,%v), want (%q,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFormatPRTitle_AndBody_Defaults(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.PullRequest.TitlePattern = "{jira_id}: {jira_title}"
	cfg.PullRequest.Template = "{jira_description}\n\n{commit_messages}\n\n* [{jira_id}]({jira_url})"

	issue := jira.Issue{Key: "APP-7", Summary: "Add OAuth2", IssueType: "Feature", URL: "https://jira/browse/APP-7", Description: "h1. Title\n\n* a\n* b"}
	title := formatPRTitle(cfg, issue)
	if !strings.HasPrefix(title, "APP-7:") {
		t.Fatalf("unexpected title: %q", title)
	}
	body, err := formatPRBody(cfg, &fakeGitInfo{commits: []string{"Implement", "Tests"}}, "main", issue)
	if err != nil {
		t.Fatalf("formatPRBody error: %v", err)
	}
	if !strings.Contains(body, "- Implement") || !strings.Contains(body, "- Tests") {
		t.Fatalf("commit messages not rendered: %q", body)
	}
	if !strings.Contains(body, "# Title") { // converted heading
		t.Fatalf("jira description not converted: %q", body)
	}
	if !strings.Contains(body, "[APP-7](https://jira/browse/APP-7)") {
		t.Fatalf("jira link not present: %q", body)
	}
}

func TestCreatePRWithDeps_Success_Overrides(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.PullRequest.Draft = true
	cfg.PullRequest.BaseBranch = "develop"
	cfg.PullRequest.TitlePattern = "{jira_id}: {jira_title}"

	jc := &fakeJiraGetter{issue: &jira.Issue{Key: "PRJ-1", Summary: "Do thing", IssueType: "Task", URL: "u"}}
	gu := &fakeGitInfo{branch: "feature/PRJ-1-thing", commits: []string{"one"}}
	gh := &fakeGH{}

	// Override base via flag and force non-draft
	url, err := createPRWithDeps(cfg, jc, gu, gh, "main", true)
	if err != nil {
		t.Fatalf("createPRWithDeps error: %v", err)
	}
	if url == "" {
		t.Fatal("expected PR url")
	}
	if gh.got.base != "main" {
		t.Fatalf("expected base main, got %q", gh.got.base)
	}
	if gh.got.draft {
		t.Fatalf("expected non-draft due to override, got draft=true")
	}
	if gh.got.head != gu.branch {
		t.Fatalf("unexpected head: %q", gh.got.head)
	}
	if !strings.HasPrefix(gh.got.title, "PRJ-1:") {
		t.Fatalf("unexpected title: %q", gh.got.title)
	}
}
