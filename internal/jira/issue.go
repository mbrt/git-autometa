package jira

import (
	"strings"

	"git-autometa/internal/markdown"
)

// Issue mirrors essential fields from JIRA responses we care about.
type Issue struct {
	Key         string
	Summary     string
	Description string
	IssueType   string
	Status      string
	Assignee    string
	URL         string
}

// DescriptionMarkdown converts the JIRA markup to Markdown.
func (i *Issue) DescriptionMarkdown() string {
	return markdown.ConvertJiraToMarkdown(i.Description)
}

// SlugifyTitle returns a basic slugified title limited to maxLength.
func (i *Issue) SlugifyTitle(maxLength int) string {
	if i.Summary == "" {
		return ""
	}
	s := strings.ToLower(i.Summary)
	s = strings.ReplaceAll(s, " ", "-")
	// trim to letters, numbers, dashes only (basic scaffolding)
	cleaned := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned = append(cleaned, r)
		}
	}
	if maxLength > 0 && len(cleaned) > maxLength {
		cleaned = cleaned[:maxLength]
	}
	return string(cleaned)
}
