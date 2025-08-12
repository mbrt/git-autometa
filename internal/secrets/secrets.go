package secrets

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

// A single keyring service to store all secrets for git-autometa.
// Keys within the service are namespaced (e.g., "jira:<email>").
const serviceName = "git-autometa"

func jiraAccountKey(email string) string {
	return "jira:" + email
}

// GetJiraToken retrieves the Jira API token for the provided email.
func GetJiraToken(email string) (string, error) {
	token, err := keyring.Get(serviceName, jiraAccountKey(email))
	if err != nil {
		return "", fmt.Errorf("secrets: unable to get jira token: %w", err)
	}
	if token == "" {
		return "", errors.New("secrets: empty jira token in keyring")
	}
	return token, nil
}

// SetJiraToken stores the Jira API token for the provided email
// in the consolidated keyring service.
func SetJiraToken(email, token string) error {
	if token == "" {
		return errors.New("secrets: empty jira token provided")
	}
	return keyring.Set(serviceName, jiraAccountKey(email), token)
}
