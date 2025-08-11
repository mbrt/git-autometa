package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appconfig "git-autometa/internal/config"
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
		if r.URL.Path != "/rest/api/2/myself" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("missing Authorization header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"self":"ok"}`))
	})
	defer ts.Close()

	if err := client.TestConnection(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestTestConnection_ErrorStatus(t *testing.T) {
	client, ts := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	})
	defer ts.Close()

	if err := client.TestConnection(); err == nil {
		t.Fatalf("expected error, got nil")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	got := issues[0]
	if got.Key != "PROJ-1" || got.Summary != "Fix bug" || got.IssueType != "Bug" || got.Status != "In Progress" || got.Assignee != "Jane" {
		t.Fatalf("unexpected issue content: %+v", got)
	}
	if got.URL == "" {
		t.Fatalf("expected URL to be set")
	}
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
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iss == nil || iss.Key != "PROJ-2" || iss.IssueType != "Story" || iss.Assignee != "John" {
		t.Fatalf("unexpected issue: %+v", iss)
	}
	if iss.URL == "" {
		t.Fatalf("expected URL to be set")
	}
}
