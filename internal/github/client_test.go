package github

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		require.Equal(t, "gh", name)
		require.GreaterOrEqual(t, len(args), 2)
		require.Equal(t, "auth", args[0])
		require.Equal(t, "status", args[1])
		return "Logged in", "", nil
	})
	require.NoError(t, c.TestConnection())
}

func TestTestConnection_Error(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "not logged in", errors.New("exit 1")
	})
	err := c.TestConnection()
	require.Error(t, err)
}

func TestCreatePullRequest_JSONOutput(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		// Verify flags contain repo scoping and json
		joined := fmt.Sprint(args)
		assert.Contains(t, joined, "--json")
		assert.Contains(t, joined, "--repo acme/project")
		return `{"url":"https://github.com/acme/project/pull/42"}`, "", nil
	})
	url, err := c.CreatePullRequest("Title", "Body", "feature/x", "main", true)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/acme/project/pull/42", url)
}

func TestCreatePullRequest_PropagatesError(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "creation failed", errors.New("exit 1")
	})
	_, err := c.CreatePullRequest("T", "B", "h", "b", false)
	require.Error(t, err)
}

func TestCreatePullRequest_RequiresTitle(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		require.Fail(t, "runner should not be called when title is empty")
		return "", "", nil
	})
	_, err := c.CreatePullRequest("", "B", "h", "b", false)
	require.Error(t, err)
}

func TestListPullRequests_JSON(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		require.Equal(t, "gh", name)
		require.GreaterOrEqual(t, len(args), 2)
		require.Equal(t, "pr", args[0])
		require.Equal(t, "list", args[1])
		data := `[
            {"number": 1, "title": "Fix bug", "url": "https://x/pr/1", "headRefName": "feat/x", "baseRefName": "main"},
            {"number": 2, "title": "Feat y", "url": "https://x/pr/2", "headRefName": "feat/y", "baseRefName": "develop"}
        ]`
		return data, "", nil
	})
	prs, err := c.ListPullRequests("open", 10)
	require.NoError(t, err)
	require.Len(t, prs, 2)
	assert.Equal(t, 1, prs[0].Number)
	assert.Equal(t, "feat/y", prs[1].HeadRefName)
}

func TestListPullRequests_Error(t *testing.T) {
	c := newTestClientWithRunner(t, func(ctx context.Context, name string, args ...string) (string, string, error) {
		return "", "boom", errors.New("exit 1")
	})
	_, err := c.ListPullRequests("open", 5)
	require.Error(t, err)
}
