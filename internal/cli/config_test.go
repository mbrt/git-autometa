package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "git-autometa/internal/config"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

// Helper to isolate XDG config in a temp directory for each test
func withTempXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Override both env and xdg package cached value
	t.Setenv("XDG_CONFIG_HOME", dir)
	prev := xdg.ConfigHome
	xdg.ConfigHome = dir
	t.Cleanup(func() { xdg.ConfigHome = prev })
	return dir
}

// Helper to override CLI owner/repo flags and restore them afterwards
func withOwnerRepo(t *testing.T, owner, repo string) {
	t.Helper()
	prevOwner, prevRepo := repoOwner, repoName
	repoOwner, repoName = owner, repo
	t.Cleanup(func() { repoOwner, repoName = prevOwner, prevRepo })
}

func TestRunConfigShowGlobal_Found_PrintsContentWithTrailingNewline(t *testing.T) {
	_ = withTempXDG(t)
	path := appconfig.GlobalConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	content := "jira:\n  server_url: https://example.atlassian.net" // no trailing newline
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	var out bytes.Buffer
	err := runConfigShowGlobal(&out)
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "Global config file:")
	assert.Contains(t, s, path)
	assert.Contains(t, s, content)
	assert.True(t, strings.HasSuffix(s, "\n"))
}

func TestRunConfigShowRepo_NotFound_FallbacksToGlobal(t *testing.T) {
	_ = withTempXDG(t)
	// Prepare a global file so fallback shows something deterministic
	gpath := appconfig.GlobalConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(gpath), 0o700))
	require.NoError(t, os.WriteFile(gpath, []byte("github:\n  owner: glob\n"), 0o600))

	withOwnerRepo(t, "acme", "tool")

	var out bytes.Buffer
	err := runConfigShowRepo(&out)
	require.NoError(t, err)
	s := out.String()
	// First line indicates repo config not found and that it will show global
	assert.Contains(t, s, "Repo config not found")
	assert.Contains(t, s, "Showing global config instead")
	// And then content from global should appear
	assert.Contains(t, s, "Global config file:")
}

func TestRunConfigShowRepo_Found_PrintsContent(t *testing.T) {
	_ = withTempXDG(t)
	withOwnerRepo(t, "octo", "repo")
	rpath := appconfig.RepoConfigPath("octo", "repo")
	require.NoError(t, os.MkdirAll(filepath.Dir(rpath), 0o700))
	content := "git:\n  branch_pattern: feat/{jira_id}\n"
	require.NoError(t, os.WriteFile(rpath, []byte(content), 0o600))

	var out bytes.Buffer
	err := runConfigShowRepo(&out)
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "Repo config file (octo/repo):")
	assert.Contains(t, s, rpath)
	assert.Contains(t, s, content)
}

func TestRunConfigShowEffective_MergesAndPrintsYAML(t *testing.T) {
	_ = withTempXDG(t)
	withOwnerRepo(t, "acme", "app")

	// Global sets some defaults
	gpath := appconfig.GlobalConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(gpath), 0o700))
	require.NoError(t, os.WriteFile(gpath, []byte("git:\n  branch_pattern: feature/{jira_id}-{jira_title}\n  max_branch_length: 40\npull_request:\n  base_branch: main\n  draft: true\n"), 0o600))
	// Repo overrides some
	rpath := appconfig.RepoConfigPath("acme", "app")
	require.NoError(t, os.MkdirAll(filepath.Dir(rpath), 0o700))
	require.NoError(t, os.WriteFile(rpath, []byte("git:\n  max_branch_length: 80\npull_request:\n  base_branch: develop\n"), 0o600))

	var out bytes.Buffer
	err := runConfigShowEffective(&out)
	require.NoError(t, err)

	// Parse YAML back to struct and verify precedence
	var got appconfig.Config
	require.NoError(t, yaml.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, 80, got.Git.MaxBranchLength)
	assert.Equal(t, "feature/{jira_id}-{jira_title}", got.Git.BranchPattern)
	assert.Equal(t, "develop", got.PullRequest.BaseBranch)
	assert.True(t, got.PullRequest.Draft)
}

func TestRunConfigEditGlobal_UpdatesAndWrites_NoToken(t *testing.T) {
	_ = withTempXDG(t)
	// No existing global file; defaults will be shown. Provide new values.
	in := bytes.NewBufferString(strings.Join([]string{
		"https://acme.atlassian.net",
		"dev@acme.io",
		"", // token: skip storing
		"",
	}, "\n"))
	var out bytes.Buffer

	err := runConfigEditGlobal(in, &out)
	require.NoError(t, err)

	path := appconfig.GlobalConfigPath()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got appconfig.Config
	require.NoError(t, yaml.Unmarshal(data, &got))
	assert.Equal(t, "https://acme.atlassian.net", got.Jira.ServerURL)
	assert.Equal(t, "dev@acme.io", got.Jira.Email)

	s := out.String()
	assert.Contains(t, s, "Saved global configuration to ")
	// No token set, so no keyring message
	assert.NotContains(t, s, "Stored JIRA token")
}

func TestRunConfigEditRepo_UpdatesAndWrites(t *testing.T) {
	_ = withTempXDG(t)
	withOwnerRepo(t, "acme", "svc")

	// Prepare a global to provide defaults for prompts
	gpath := appconfig.GlobalConfigPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(gpath), 0o700))
	require.NoError(t, os.WriteFile(gpath, []byte("git:\n  branch_pattern: feature/{jira_id}-{jira_title}\n  max_branch_length: 50\npull_request:\n  title_pattern: '{jira_id}: {jira_title}'\n  base_branch: main\n  draft: true\n"), 0o600))

	// Inputs: override max length, title pattern, and draft yes; keep others default
	in := bytes.NewBufferString(strings.Join([]string{
		"",                 // branch pattern -> keep default
		"70",               // max length
		"{jira_id} - work", // title pattern
		"",                 // base branch -> keep default
		"yes",              // draft true
		"",
	}, "\n"))
	var out bytes.Buffer

	err := runConfigEditRepo(in, &out)
	require.NoError(t, err)

	rpath := appconfig.RepoConfigPath("acme", "svc")
	data, err := os.ReadFile(rpath)
	require.NoError(t, err)
	var overrides appconfig.Config
	require.NoError(t, yaml.Unmarshal(data, &overrides))

	assert.Equal(t, 70, overrides.Git.MaxBranchLength)
	assert.Equal(t, "{jira_id} - work", overrides.PullRequest.TitlePattern)
	assert.True(t, overrides.PullRequest.Draft)
	// Not overridden fields should be zero-values in overrides file
	assert.Equal(t, "", overrides.Git.BranchPattern)
	assert.Equal(t, "", overrides.PullRequest.BaseBranch)

	assert.Contains(t, out.String(), "Saved repository configuration for acme/svc")
}

func TestRunConfigEditRepo_NoOwnerRepo_Error(t *testing.T) {
	_ = withTempXDG(t)
	withOwnerRepo(t, "", "")
	// Move to a non-git temporary directory so autodetection via git fails
	cwd, _ := os.Getwd()
	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	var out bytes.Buffer
	err := runConfigEditRepo(bytes.NewBuffer(nil), &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository owner/repo not detected")
}

func TestParseGitHubOwnerRepo_Variants(t *testing.T) {
	cases := []struct {
		in    string
		owner string
		repo  string
		ok    bool
	}{
		{"https://github.com/owner/repo.git", "owner", "repo", true},
		{"https://github.com/owner/repo", "owner", "repo", true},
		{"git@github.com:owner/repo.git", "owner", "repo", true},
		{"ssh://git@github.com/owner/repo", "owner", "repo", true},
		{"https://gitlab.com/owner/repo", "", "", false},
		{"invalid", "", "", false},
	}
	for _, tc := range cases {
		gotOwner, gotRepo, ok := parseGitHubOwnerRepo(tc.in)
		assert.Equal(t, tc.ok, ok, "ok mismatch for %q", tc.in)
		if tc.ok {
			assert.Equal(t, tc.owner, gotOwner)
			assert.Equal(t, tc.repo, gotRepo)
		}
	}
}
