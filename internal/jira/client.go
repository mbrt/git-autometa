package jira

import (
	appconfig "git-autometa/internal/config"
)

// Client is a placeholder for a future JIRA client.
type Client struct {
	serverURL string
	email     string
}

func NewClient(cfg *appconfig.Config) *Client {
	return &Client{
		serverURL: cfg.Jira.ServerURL,
		email:     cfg.Jira.Email,
	}
}

func (c *Client) TestConnection() error {
	// TODO: implement connectivity check
	return nil
}

func (c *Client) SearchMyIssues(limit int) ([]Issue, error) {
	// TODO: implement JQL search
	return []Issue{}, nil
}

func (c *Client) GetIssue(key string) (*Issue, error) {
	// TODO: implement get issue
	return nil, nil
}
