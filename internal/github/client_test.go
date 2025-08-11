package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	appconfig "git-autometa/internal/config"
)

func newTestClientWithRunner(t *testing.T, run runnerFunc) Client {
	t.Helper()
	cfg := appconfig.Config{
		GitHub: appconfig.GitHubConfig{Owner: "acme", Repo: "project"},
	}
	c := NewClient(cfg)
	c.run = run
	return c
}

func TestTestConnection_OK(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		if name != "gh" || len(args) < 2 || args[0] != "auth" || args[1] != "status" {
			t.Fatalf("unexpected invocation: %s %v", name, args)
		}
		return "Logged in", "", nil
	})
	if err := c.TestConnection(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTestConnection_Error(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "not logged in", errors.New("exit 1")
	})
	if err := c.TestConnection(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCreatePullRequest_JSONOutput(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		// Verify flags contain repo scoping and json
		joined := fmt.Sprint(args)
		if want := "--json"; !contains(joined, want) {
			t.Fatalf("expected %s in args: %v", want, args)
		}
		if want := "--repo acme/project"; !contains(joined, want) {
			t.Fatalf("expected %s in args: %v", want, args)
		}
		return `{"url":"https://github.com/acme/project/pull/42"}`, "", nil
	})
	url, err := c.CreatePullRequest("Title", "Body", "feature/x", "main", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/acme/project/pull/42" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestCreatePullRequest_NonJSONFallback(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "https://github.com/acme/project/pull/99\n", "", nil
	})
	url, err := c.CreatePullRequest("Title", "Body", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/acme/project/pull/99" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestCreatePullRequest_PropagatesError(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "creation failed", errors.New("exit 1")
	})
	if _, err := c.CreatePullRequest("T", "B", "h", "b", false); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCreatePullRequest_RequiresTitle(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		t.Fatalf("runner should not be called when title is empty")
		return "", "", nil
	})
	if _, err := c.CreatePullRequest("", "B", "h", "b", false); err == nil {
		t.Fatalf("expected error for empty title")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestListPullRequests_JSON(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		if name != "gh" || len(args) == 0 || args[0] != "pr" || args[1] != "list" {
			t.Fatalf("unexpected invocation: %s %v", name, args)
		}
		data := `[
            {"number": 1, "title": "Fix bug", "url": "https://x/pr/1", "headRefName": "feat/x", "baseRefName": "main"},
            {"number": 2, "title": "Feat y", "url": "https://x/pr/2", "headRefName": "feat/y", "baseRefName": "develop"}
        ]`
		return data, "", nil
	})
	prs, err := c.ListPullRequests("open", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 || prs[0].Number != 1 || prs[1].HeadRefName != "feat/y" {
		t.Fatalf("unexpected prs: %+v", prs)
	}
}

func TestListPullRequests_Error(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "boom", errors.New("exit 1")
	})
	if _, err := c.ListPullRequests("open", 5); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
