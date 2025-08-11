package cli

import (
	"bytes"
	"testing"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/jira"
)

func TestFormatBranchName_DefaultPattern(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.Git.BranchPattern = "feature/{jira_id}-{jira_title}"
	cfg.Git.MaxBranchLength = 100

	issue := jira.Issue{
		Key:       "PROJ-123",
		Summary:   "Fix Login Validation Bug",
		IssueType: "Bug",
	}

	got := formatBranchName(cfg, issue)
	want := "feature-PROJ-123-fix-login-validation-bug"
	if got != want {
		t.Fatalf("formatBranchName() = %q, want %q", got, want)
	}
}

func TestFormatBranchName_CustomPatternAndType(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.Git.BranchPattern = "{jira_type}/{jira_id}"
	cfg.Git.MaxBranchLength = 100

	issue := jira.Issue{
		Key:       "APP-7",
		Summary:   "Add OAuth2",
		IssueType: "Feature",
	}

	got := formatBranchName(cfg, issue)
	want := "feature-APP-7"
	if got != want {
		t.Fatalf("formatBranchName() = %q, want %q", got, want)
	}
}

func TestFormatBranchName_MaxLength(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.Git.BranchPattern = "feature/{jira_id}-{jira_title}"
	cfg.Git.MaxBranchLength = 20

	issue := jira.Issue{
		Key:     "PROJ-10",
		Summary: "This is a very long title that should be truncated",
	}

	got := formatBranchName(cfg, issue)
	if len(got) > cfg.Git.MaxBranchLength {
		t.Fatalf("branch length %d exceeds max %d: %q", len(got), cfg.Git.MaxBranchLength, got)
	}
}

func TestSanitizeBranchName_RemovesDisallowedAndCollapses(t *testing.T) {
	in := "feat//weird^name..with*[chars]? and spaces"
	got := sanitizeBranchName(in)
	// Ensure forbidden tokens replaced and multiple dashes/slashes collapsed
	if got == in {
		t.Fatalf("sanitizeBranchName did not change input: %q", got)
	}
	if containsAny(got, []string{"^", ":", "?", "*", "[", ".."}) {
		t.Fatalf("sanitizeBranchName left disallowed chars in: %q", got)
	}
}

func containsAny(s string, tokens []string) bool {
	for _, tok := range tokens {
		if idx := indexOf(s, tok); idx >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, substr string) int {
	// small helper to keep imports minimal
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// --- fakes for testing startWorkWithDeps ---

type fakeJira struct {
	issues []jira.Issue
	issue  *jira.Issue
	err    error
}

func (f *fakeJira) SearchMyIssues(limit int) ([]jira.Issue, error) { return f.issues, f.err }
func (f *fakeJira) GetIssue(key string) (*jira.Issue, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.issue != nil {
		return f.issue, nil
	}
	return &jira.Issue{Key: key, Summary: "S", IssueType: "Task"}, nil
}

type fakeGit struct {
	prepared string
	pushed   string
	err      error
}

func (g *fakeGit) PrepareWorkBranch(desiredBranchName string) (string, error) {
	g.prepared = desiredBranchName
	if g.err != nil {
		return "", g.err
	}
	return desiredBranchName, nil
}
func (g *fakeGit) PushBranch(branchName string) error {
	g.pushed = branchName
	return g.err
}

func TestStartWorkWithDeps_ArgFlow_NoPush(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.Git.BranchPattern = "feature/{jira_id}-{jira_title}"
	cfg.Git.MaxBranchLength = 60

	fj := &fakeJira{issue: &jira.Issue{Key: "P-1", Summary: "Login fix", IssueType: "Bug"}}
	fg := &fakeGit{}

	var out bytes.Buffer
	err := startWorkWithDeps([]string{"P-1"}, cfg, fj, fg, bytes.NewBuffer(nil), &out, false)
	if err != nil {
		t.Fatalf("startWorkWithDeps error: %v", err)
	}
	if fg.prepared != "feature-P-1-login-fix" {
		t.Fatalf("unexpected prepared branch: %q", fg.prepared)
	}
	if fg.pushed != "" {
		t.Fatalf("push should not be called; got %q", fg.pushed)
	}
	if out.Len() == 0 {
		t.Fatal("expected some output")
	}
}

func TestStartWorkWithDeps_Interactive_Push(t *testing.T) {
	cfg := appconfig.DefaultConfig()
	cfg.Git.BranchPattern = "{jira_id}"

	fj := &fakeJira{issues: []jira.Issue{{Key: "X-7", Summary: "Do it"}}}
	fg := &fakeGit{}

	// Simulate selecting first issue (enter "1")
	in := bytes.NewBufferString("1\n")
	var out bytes.Buffer

	err := startWorkWithDeps(nil, cfg, fj, fg, in, &out, true)
	if err != nil {
		t.Fatalf("startWorkWithDeps error: %v", err)
	}
	if fg.prepared != "X-7" {
		t.Fatalf("unexpected prepared branch: %q", fg.prepared)
	}
	if fg.pushed != "X-7" {
		t.Fatalf("expected push of X-7; got %q", fg.pushed)
	}
}
