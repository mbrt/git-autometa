package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runCmd(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "command failed: %s %s\n%s", name, strings.Join(args, " "), string(out))
	return string(out)
}

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	// basic identity for commits
	runCmd(t, dir, "git", "config", "user.name", "Test User")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")

	// create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, dir, "git", "add", "README.md")
	runCmd(t, dir, "git", "commit", "-m", "initial commit")

	// ensure base branch named 'main'
	runCmd(t, dir, "git", "checkout", "-B", "main")
	return dir
}

func TestPrepareWorkBranch_CreatesAndIncrements(t *testing.T) {
	repoDir := initTempRepo(t)
	git := Git{WorkDir: repoDir, MainBranch: "main"}

	// First creation should use desired name
	name, err := git.PrepareWorkBranch("feature/test")
	require.NoError(t, err)
	assert.Equal(t, "feature/test", name)
	cur, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature/test", cur)

	// Second creation with same base should increment
	name2, err := git.PrepareWorkBranch("feature/test")
	require.NoError(t, err)
	assert.Equal(t, "feature/test-2", name2)
	cur2, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feature/test-2", cur2)
}

func TestPushBranch_NoOrigin(t *testing.T) {
	repoDir := initTempRepo(t)
	git := Git{WorkDir: repoDir, MainBranch: "main"}

	_, err := git.PrepareWorkBranch("feat/push-no-origin")
	require.NoError(t, err)
	err = git.PushBranch("feat/push-no-origin")
	require.Error(t, err)
}

func TestPushBranch_WithOriginAndRemoteURL(t *testing.T) {
	repoDir := initTempRepo(t)
	git := Git{WorkDir: repoDir, MainBranch: "main"}

	// Create bare remote repo
	remoteDir := t.TempDir()
	runCmd(t, remoteDir, "git", "init", "--bare")

	// Add origin remote to working repo
	runCmd(t, repoDir, "git", "remote", "add", "origin", remoteDir)

	_, err := git.PrepareWorkBranch("feat/push-origin")
	require.NoError(t, err)
	err = git.PushBranch("feat/push-origin")
	require.NoError(t, err)

	url, err := git.GetRemoteURL("origin")
	require.NoError(t, err)
	assert.Equal(t, remoteDir, strings.TrimSpace(url))
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir := initTempRepo(t)
	git := Git{WorkDir: repoDir, MainBranch: "main"}

	_, err := git.PrepareWorkBranch("feat/current")
	require.NoError(t, err)
	cur, err := git.GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "feat/current", cur)
}

func TestGetCommitMessagesForPR_CleansTags(t *testing.T) {
	repoDir := initTempRepo(t)
	git := Git{WorkDir: repoDir, MainBranch: "main"}

	_, err := git.PrepareWorkBranch("feat/commits")
	require.NoError(t, err)

	// two commits with jira-like subjects
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, repoDir, "git", "add", "file.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "[PROJ-1] Implement feature")

	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, repoDir, "git", "add", "file.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "PROJ-1: Add tests")

	msgs, err := git.GetCommitMessagesForPR("main")
	require.NoError(t, err)
	got := strings.Join(msgs, "|")
	// git log returns newest first by default
	want := "Add tests|Implement feature"
	assert.Equal(t, want, got)
}
