package github

import (
	appconfig "git-autometa/internal/config"
)

// Client is a placeholder for a future GitHub client delegating to gh CLI.
type Client struct{}

func NewClient(cfg *appconfig.Config) *Client {
	_ = cfg
	return &Client{}
}

func (c *Client) TestConnection() error {
	// TODO: implement connectivity checks via gh CLI
	return nil
}

// CreatePullRequest is a placeholder that returns a dummy URL.
func (c *Client) CreatePullRequest(title, body, head, base string, draft bool) (string, error) {
	// TODO: implement via gh CLI
	return "https://github.com/owner/repo/pull/1", nil
}
