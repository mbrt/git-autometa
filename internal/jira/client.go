package jira

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/secrets"
)

// Note: secrets are retrieved via the centralized secrets package.

// Client provides minimal Jira REST API access required by the CLI.
type Client struct {
	serverURL  string
	email      string
	httpClient *http.Client
	token      string
}

func NewClient(cfg appconfig.Config, token string) Client {
	return Client{
		serverURL: strings.TrimRight(cfg.Jira.ServerURL, "/"),
		email:     cfg.Jira.Email,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		token: token,
	}
}

// NewClientWithKeyring is a convenience constructor that reads the token once from keyring.
func NewClientWithKeyring(cfg appconfig.Config) (Client, error) {
	if cfg.Jira.Email == "" {
		return Client{}, errors.New("jira: missing email in config")
	}
	token, err := secrets.GetJiraToken(cfg.Jira.Email)
	if err != nil {
		return Client{}, fmt.Errorf("jira: unable to load API token from keyring for %s: %w", cfg.Jira.Email, err)
	}
	return NewClient(cfg, token), nil
}

// TestConnection verifies credentials by calling Jira's /myself endpoint.
func (c Client) TestConnection() error {
	req, err := c.newRequest(http.MethodGet, "/rest/api/2/myself", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("jira: test connection failed: %s: %s", resp.Status, string(body))
	}
	return nil
}

// SearchMyIssues searches for issues assigned to the current user, excluding Done, ordered by last update.
func (c Client) SearchMyIssues(limit int) ([]Issue, error) {
	q := url.Values{}
	q.Set("jql", "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC")
	q.Set("maxResults", fmt.Sprintf("%d", limit))
	q.Set("fields", "summary,description,issuetype,status,assignee")

	req, err := c.newRequest(http.MethodGet, "/rest/api/2/search", q)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("jira: search issues failed: %s: %s", resp.Status, string(body))
	}

	var payload struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary     string `json:"summary"`
				Description string `json:"description"`
				IssueType   struct {
					Name string `json:"name"`
				} `json:"issuetype"`
				Status struct {
					Name string `json:"name"`
				} `json:"status"`
				Assignee *struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]Issue, 0, len(payload.Issues))
	for _, it := range payload.Issues {
		assignee := ""
		if it.Fields.Assignee != nil {
			assignee = it.Fields.Assignee.DisplayName
		}
		out = append(out, Issue{
			Key:         it.Key,
			Summary:     it.Fields.Summary,
			Description: it.Fields.Description,
			IssueType:   it.Fields.IssueType.Name,
			Status:      it.Fields.Status.Name,
			Assignee:    assignee,
			URL:         c.issueURL(it.Key),
		})
	}
	return out, nil
}

// GetIssue fetches a single issue by key.
func (c Client) GetIssue(key string) (*Issue, error) {
	if key == "" {
		return nil, errors.New("jira: empty issue key")
	}

	q := url.Values{}
	q.Set("fields", "summary,description,issuetype,status,assignee")
	endpoint := "/rest/api/2/issue/" + url.PathEscape(key)
	req, err := c.newRequest(http.MethodGet, endpoint, q)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("jira: get issue %s failed: %s: %s", key, resp.Status, string(body))
	}

	var payload struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description string `json:"description"`
			IssueType   struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	assignee := ""
	if payload.Fields.Assignee != nil {
		assignee = payload.Fields.Assignee.DisplayName
	}
	result := &Issue{
		Key:         payload.Key,
		Summary:     payload.Fields.Summary,
		Description: payload.Fields.Description,
		IssueType:   payload.Fields.IssueType.Name,
		Status:      payload.Fields.Status.Name,
		Assignee:    assignee,
		URL:         c.issueURL(payload.Key),
	}
	return result, nil
}

func (c Client) newRequest(method, p string, query url.Values) (*http.Request, error) {
	base, err := url.Parse(c.serverURL)
	if err != nil {
		return nil, fmt.Errorf("jira: invalid server URL: %w", err)
	}
	// Ensure path is joined correctly
	base.Path = path.Join(base.Path, p)
	if query != nil {
		base.RawQuery = query.Encode()
	}
	req, err := http.NewRequest(method, base.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	// Jira Cloud uses basic auth with email:token
	if c.email == "" {
		return nil, errors.New("jira: missing email in client")
	}
	if c.token == "" {
		return nil, errors.New("jira: missing token in client")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))
	req.Header.Set("Authorization", "Basic "+auth)
	return req, nil
}

func (c Client) issueURL(key string) string {
	return strings.TrimRight(c.serverURL, "/") + "/browse/" + key
}
