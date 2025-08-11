package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCmd(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s: %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
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
	git := NewUtilsWithWorkDir(repoDir)

	// First creation should use desired name
	name, err := git.PrepareWorkBranch("feature/test")
	if err != nil {
		t.Fatalf("PrepareWorkBranch error: %v", err)
	}
	if name != "feature/test" {
		t.Fatalf("expected branch name 'feature/test', got %q", name)
	}
	cur, err := git.GetCurrentBranch()
	if err != nil || cur != "feature/test" {
		t.Fatalf("expected to be on 'feature/test', got %q, err=%v", cur, err)
	}

	// Second creation with same base should increment
	name2, err := git.PrepareWorkBranch("feature/test")
	if err != nil {
		t.Fatalf("PrepareWorkBranch (second) error: %v", err)
	}
	if name2 != "feature/test-2" {
		t.Fatalf("expected branch name 'feature/test-2', got %q", name2)
	}
	cur2, err := git.GetCurrentBranch()
	if err != nil || cur2 != "feature/test-2" {
		t.Fatalf("expected to be on 'feature/test-2', got %q, err=%v", cur2, err)
	}
}

func TestPushBranch_NoOrigin(t *testing.T) {
	repoDir := initTempRepo(t)
	git := NewUtilsWithWorkDir(repoDir)

	if _, err := git.PrepareWorkBranch("feat/push-no-origin"); err != nil {
		t.Fatalf("prepare branch: %v", err)
	}
	if err := git.PushBranch("feat/push-no-origin"); err == nil {
		t.Fatalf("expected error when pushing without origin remote")
	}
}

func TestPushBranch_WithOriginAndRemoteURL(t *testing.T) {
	repoDir := initTempRepo(t)
	git := NewUtilsWithWorkDir(repoDir)

	// Create bare remote repo
	remoteDir := t.TempDir()
	runCmd(t, remoteDir, "git", "init", "--bare")

	// Add origin remote to working repo
	runCmd(t, repoDir, "git", "remote", "add", "origin", remoteDir)

	if _, err := git.PrepareWorkBranch("feat/push-origin"); err != nil {
		t.Fatalf("prepare branch: %v", err)
	}
	if err := git.PushBranch("feat/push-origin"); err != nil {
		t.Fatalf("push branch: %v", err)
	}

	url, err := git.GetRemoteURL("origin")
	if err != nil {
		t.Fatalf("get remote url: %v", err)
	}
	if strings.TrimSpace(url) != remoteDir {
		t.Fatalf("unexpected remote url: %q (want %q)", url, remoteDir)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir := initTempRepo(t)
	git := NewUtilsWithWorkDir(repoDir)

	if _, err := git.PrepareWorkBranch("feat/current"); err != nil {
		t.Fatalf("prepare branch: %v", err)
	}
	cur, err := git.GetCurrentBranch()
	if err != nil {
		t.Fatalf("get current branch: %v", err)
	}
	if cur != "feat/current" {
		t.Fatalf("unexpected current branch: %q", cur)
	}
}

func TestGetCommitMessagesForPR_CleansTags(t *testing.T) {
	repoDir := initTempRepo(t)
	git := NewUtilsWithWorkDir(repoDir)

	if _, err := git.PrepareWorkBranch("feat/commits"); err != nil {
		t.Fatalf("prepare branch: %v", err)
	}

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
	if err != nil {
		t.Fatalf("get commit messages: %v", err)
	}
	got := strings.Join(msgs, "|")
	// git log returns newest first by default
	want := "Add tests|Implement feature"
	if got != want {
		t.Fatalf("unexpected messages: %q (want %q)", got, want)
	}
}
