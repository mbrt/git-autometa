package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appconfig "git-autometa/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	cfg := appconfig.Config{
		Jira: appconfig.JiraConfig{
			ServerURL: ts.URL,
			Email:     "user@example.com",
		},
	}
	c := NewClient(cfg, "test-token")
	// Use the test server's default client to avoid leaking connections.
	c.httpClient = ts.Client()
	return c, ts
}

func TestTestConnection_OK(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/rest/api/2/myself", r.URL.Path)
		require.NotEmpty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"self":"ok"}`))
	})
	defer ts.Close()

	require.NoError(t, client.TestConnection())
}

func TestTestConnection_ErrorStatus(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	})
	defer ts.Close()

	err := client.TestConnection()
	require.Error(t, err)
}

func TestSearchMyIssues(t *testing.T) {
	payload := map[string]any{
		"issues": []any{
			map[string]any{
				"key": "PROJ-1",
				"fields": map[string]any{
					"summary":     "Fix bug",
					"description": "desc",
					"issuetype":   map[string]any{"name": "Bug"},
					"status":      map[string]any{"name": "In Progress"},
					"assignee":    map[string]any{"displayName": "Jane"},
				},
			},
		},
	}
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("maxResults") == "" {
			t.Fatalf("expected maxResults query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	})
	defer ts.Close()

	issues, err := client.SearchMyIssues(5)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	got := issues[0]
	assert.Equal(t, "PROJ-1", got.Key)
	assert.Equal(t, "Fix bug", got.Summary)
	assert.Equal(t, "Bug", got.IssueType)
	assert.Equal(t, "In Progress", got.Status)
	assert.Equal(t, "Jane", got.Assignee)
	assert.NotEmpty(t, got.URL)
}

func TestGetIssue(t *testing.T) {
	payload := map[string]any{
		"key": "PROJ-2",
		"fields": map[string]any{
			"summary":     "Implement feature",
			"description": "md",
			"issuetype":   map[string]any{"name": "Story"},
			"status":      map[string]any{"name": "To Do"},
			"assignee":    map[string]any{"displayName": "John"},
		},
	}
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issue/PROJ-2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	})
	defer ts.Close()

	iss, err := client.GetIssue("PROJ-2")
	require.NoError(t, err)
	require.NotNil(t, iss)
	assert.Equal(t, "PROJ-2", iss.Key)
	assert.Equal(t, "Story", iss.IssueType)
	assert.Equal(t, "John", iss.Assignee)
	assert.NotEmpty(t, iss.URL)
}
