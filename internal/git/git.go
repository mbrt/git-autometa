package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Git is a wrapper around Git operations executed in the current working directory.
type Git struct {
	// WorkDir is the filesystem path where git commands should run.
	// If empty, commands run in the current process working directory.
	WorkDir string
}

func New() *Git { return &Git{} }

// NewWithWorkDir creates a Utils bound to the provided working directory.
func NewWithWorkDir(dir string) *Git { return &Git{WorkDir: dir} }

// PrepareWorkBranch ensures the repository is up to date on the default branch (main/master)
// and creates/switches to a new work branch derived from the provided name.
// If the desired branch already exists locally or remotely, it will automatically
// append an incrementing numeric suffix (e.g., "-2", "-3", ...) until an unused name is found.
func (g *Git) PrepareWorkBranch(desiredBranchName string) (string, error) {
	if err := g.assertGitRepo(); err != nil {
		return "", err
	}

	// Fetch remotes if any
	_, _ = runGitDir(g.WorkDir, "fetch", "--all", "-p")

	// Detect main branch: prefer main, fallback to master
	mainBranch := "main"
	hasMain := branchExistsLocallyDir(g.WorkDir, "main") || remoteBranchExistsDir(g.WorkDir, "origin", "main")
	if !hasMain {
		if branchExistsLocallyDir(g.WorkDir, "master") || remoteBranchExistsDir(g.WorkDir, "origin", "master") {
			mainBranch = "master"
		}
	}

	// Checkout main branch if it exists locally, otherwise create it tracking remote if present.
	if branchExistsLocallyDir(g.WorkDir, mainBranch) {
		if _, err := runGitDir(g.WorkDir, "checkout", mainBranch); err != nil {
			return "", err
		}
	} else if remoteBranchExistsDir(g.WorkDir, "origin", mainBranch) {
		if _, err := runGitDir(g.WorkDir, "checkout", "-b", mainBranch, fmt.Sprintf("origin/%s", mainBranch)); err != nil {
			return "", err
		}
	}

	// Pull latest if origin exists and remote branch available
	if hasRemoteDir(g.WorkDir, "origin") && remoteBranchExistsDir(g.WorkDir, "origin", mainBranch) {
		_, _ = runGitDir(g.WorkDir, "pull", "--ff-only", "origin", mainBranch)
	}

	// Compute final branch name (auto-increment if exists locally or remotely)
	finalBranchName := desiredBranchName
	for branchExistsLocallyDir(g.WorkDir, finalBranchName) || remoteBranchExistsDir(g.WorkDir, "origin", finalBranchName) {
		finalBranchName = incrementBranchName(finalBranchName)
	}

	// Create branch from main and switch to it
	args := []string{"checkout", "-b", finalBranchName}
	if branchExistsLocallyDir(g.WorkDir, mainBranch) || remoteBranchExistsDir(g.WorkDir, "origin", mainBranch) {
		args = append(args, mainBranch)
	}
	if _, err := runGitDir(g.WorkDir, args...); err != nil {
		return "", err
	}

	return finalBranchName, nil
}

// PushBranch pushes the given branch to origin and sets upstream if not set.
func (g *Git) PushBranch(branchName string) error {
	if err := g.assertGitRepo(); err != nil {
		return err
	}
	if !hasRemoteDir(g.WorkDir, "origin") {
		return errors.New("no 'origin' remote configured")
	}
	_, err := runGitDir(g.WorkDir, "push", "-u", "origin", branchName)
	return err
}

// GetCurrentBranch returns the name of the current checked-out branch.
func (g *Git) GetCurrentBranch() (string, error) {
	if err := g.assertGitRepo(); err != nil {
		return "", err
	}
	out, err := runGitDir(g.WorkDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

// GetCommitMessagesForPR returns commit subjects from baseBranch..HEAD, with leading
// JIRA tags like "[ABC-123]" or "ABC-123:" stripped from each message.
func (g *Git) GetCommitMessagesForPR(baseBranch string) ([]string, error) {
	if err := g.assertGitRepo(); err != nil {
		return nil, err
	}
	rangeSpec := fmt.Sprintf("%s..HEAD", baseBranch)
	out, err := runGitDir(g.WorkDir, "log", "--no-merges", "--pretty=%s", rangeSpec)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	// Patterns to remove leading JIRA identifiers
	squareTag := regexp.MustCompile(`^\[[A-Z][A-Z0-9]+-\d+\]\s*`)
	bareTag := regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+\s*[:\-]?\s*`)
	messages := make([]string, 0, len(lines))
	for _, line := range lines {
		cleaned := strings.TrimSpace(line)
		cleaned = squareTag.ReplaceAllString(cleaned, "")
		cleaned = bareTag.ReplaceAllString(cleaned, "")
		if cleaned != "" {
			messages = append(messages, cleaned)
		}
	}
	return messages, nil
}

// GetRemoteURL returns the URL for the given remote.
func (g *Git) GetRemoteURL(remote string) (string, error) {
	if err := g.assertGitRepo(); err != nil {
		return "", err
	}
	out, err := runGitDir(g.WorkDir, "remote", "get-url", remote)
	if err != nil {
		return "", err
	}
	return out, nil
}

// --- helpers ---

func (g *Git) assertGitRepo() error {
	_, err := runGitDir(g.WorkDir, "rev-parse", "--git-dir")
	return err
}

func runGitDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Include stderr to help callers
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func hasRemoteDir(dir, name string) bool {
	out, err := runGitDir(dir, "remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if line == name {
			return true
		}
	}
	return false
}

func branchExistsLocallyDir(dir, name string) bool {
	_, err := runGitDir(dir, "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", name))
	return err == nil
}

func remoteBranchExistsDir(dir, remote, branch string) bool {
	if !hasRemoteDir(dir, remote) {
		return false
	}
	_, err := runGitDir(dir, "ls-remote", "--exit-code", "--heads", remote, fmt.Sprintf("refs/heads/%s", branch))
	return err == nil
}

func incrementBranchName(name string) string {
	// If name already ends with -<number>, increment it; otherwise append -2
	lastDash := strings.LastIndex(name, "-")
	if lastDash == -1 {
		return name + "-2"
	}
	prefix := name[:lastDash]
	suffix := name[lastDash+1:]
	// Try to parse numeric suffix
	parsed, err := strconv.Atoi(suffix)
	if err != nil || parsed <= 1 {
		return name + "-2"
	}
	return fmt.Sprintf("%s-%d", prefix, parsed+1)
}
