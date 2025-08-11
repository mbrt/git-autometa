package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	appconfig "git-autometa/internal/config"
)

type runnerFunc func(ctx context.Context, name string, args ...string) (stdout string, stderr string, err error)

// Client provides minimal GitHub operations via the gh CLI.
type Client struct {
	owner string
	repo  string
	run   runnerFunc
}

// NewClient constructs a GitHub client. If cfg.GitHub.Owner/Repo are set, they
// will be used to scope gh operations via --repo <owner>/<repo>.
func NewClient(cfg appconfig.Config) Client {
	return Client{
		owner: strings.TrimSpace(cfg.GitHub.Owner),
		repo:  strings.TrimSpace(cfg.GitHub.Repo),
		run:   defaultRunner,
	}
}

// TestConnection verifies that gh is authenticated for GitHub.
func (c Client) TestConnection() error {
	_, stderr, err := c.run(context.Background(), "gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("github: gh auth status failed: %v: %s", err, strings.TrimSpace(stderr))
	}
	return nil
}

// CreatePullRequest creates a pull request using gh and returns the PR URL.
func (c Client) CreatePullRequest(title, body, head, base string, draft bool) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", errors.New("github: title is required")
	}
	args := []string{"pr", "create", "--title", title, "--body", body, "--json", "url"}
	if head != "" {
		args = append(args, "--head", head)
	}
	if base != "" {
		args = append(args, "--base", base)
	}
	if draft {
		args = append(args, "--draft")
	}
	if c.owner != "" && c.repo != "" {
		args = append(args, "--repo", c.owner+"/"+c.repo)
	}

	stdout, stderr, err := c.run(context.Background(), "gh", args...)
	if err != nil {
		return "", fmt.Errorf("github: gh pr create failed: %v: %s", err, strings.TrimSpace(stderr))
	}

	// gh --json url returns a small JSON object {"url":"..."}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&payload); err != nil {
		// If output isn't JSON (older gh or unexpected), fallback to trimming stdout
		trimmed := strings.TrimSpace(stdout)
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			return trimmed, nil
		}
		return "", fmt.Errorf("github: unable to parse gh output: %w; output: %s", err, trimmed)
	}
	if payload.URL == "" {
		return "", errors.New("github: gh returned empty URL")
	}
	return payload.URL, nil
}

// PullRequest represents minimal data about a PR used by the CLI.
type PullRequest struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
}

// ListPullRequests lists pull requests via gh.
// State can be one of: open, closed, merged, all. Limit <= 0 means gh default.
func (c Client) ListPullRequests(state string, limit int) ([]PullRequest, error) {
	args := []string{"pr", "list", "--json", "number,title,url,headRefName,baseRefName"}
	if state != "" {
		args = append(args, "--state", state)
	}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", limit))
	}
	if c.owner != "" && c.repo != "" {
		args = append(args, "--repo", c.owner+"/"+c.repo)
	}
	stdout, stderr, err := c.run(context.Background(), "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("github: gh pr list failed: %v: %s", err, strings.TrimSpace(stderr))
	}
	var prs []PullRequest
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&prs); err != nil {
		return nil, fmt.Errorf("github: unable to parse gh pr list output: %w", err)
	}
	return prs, nil
}

func defaultRunner(ctx context.Context, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	// Keep a reasonable timeout to avoid hanging if gh prompts for input
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, name, args...)
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}
	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}
