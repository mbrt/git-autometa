package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// Config mirrors the keys from example-config.yaml and the Python architecture.
type Config struct {
	Jira        JiraConfig        `yaml:"jira"`
	GitHub      GitHubConfig      `yaml:"github"`
	Git         GitConfig         `yaml:"git"`
	PullRequest PullRequestConfig `yaml:"pull_request"`
}

type JiraConfig struct {
	ServerURL string `yaml:"server_url"`
	Email     string `yaml:"email"`
}

type GitHubConfig struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

type GitConfig struct {
	BranchPattern   string `yaml:"branch_pattern"`
	MaxBranchLength int    `yaml:"max_branch_length"`
}

type PullRequestConfig struct {
	TitlePattern string `yaml:"title_pattern"`
	Draft        bool   `yaml:"draft"`
	BaseBranch   string `yaml:"base_branch"`
	Template     string `yaml:"template"`
}

// DefaultConfig returns sensible defaults aligned with example-config.yaml.
func DefaultConfig() Config {
	return Config{
		Jira: JiraConfig{
			ServerURL: "https://your-company.atlassian.net",
			Email:     "",
		},
		GitHub: GitHubConfig{
			Owner: "",
			Repo:  "",
		},
		Git: GitConfig{
			BranchPattern:   "feature/{jira_id}-{jira_title}",
			MaxBranchLength: 50,
		},
		PullRequest: PullRequestConfig{
			TitlePattern: "{jira_id}: {jira_title}",
			Draft:        true,
			BaseBranch:   "main",
			Template: `# What this Pull Request does/why we need it

{jira_description}

{commit_messages}

## What type of PR is this?

feature

## Relevant links

* [{jira_id}]({jira_url})
`,
		},
	}
}

// LoadConfigForRepo loads the configuration for a given repository.
// It merges the global config with the repo-specific config.
func LoadConfigForRepo(owner, repo string) (Config, error) {
	return LoadEffectiveConfig(
		GlobalConfigPath(),
		RepoConfigPath(owner, repo),
	)
}

// LoadEffectiveConfig loads the configuration using priority order.
// The last path takes precedence over the previous ones.
func LoadEffectiveConfig(paths ...string) (Config, error) {
	cfg := DefaultConfig()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, err
		}
		// Unmarshal directly into cfg.
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// GlobalConfigPath returns the global config path using XDG base directories.
func GlobalConfigPath() string {
	// Use XDG config directory, defaulting per the library behavior.
	// Example on Linux: ~/.config/git-autometa/config.yaml
	path, err := xdg.ConfigFile(filepath.Join("git-autometa", "config.yaml"))
	if err != nil {
		// Fall back to a relative file if XDG resolution fails
		return "config.yaml"
	}
	return path
}

// RepoConfigPath returns the repo-specific config path using XDG base directories.
func RepoConfigPath(owner, repo string) string {
	rel := filepath.Join("git-autometa", "repositories", owner+"_"+repo+".yaml")
	path, err := xdg.ConfigFile(rel)
	if err != nil {
		return filepath.Join("repositories", owner+"_"+repo+".yaml")
	}
	return path
}
