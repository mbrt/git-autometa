package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config mirrors the keys from example-config.yaml and the Python architecture.
type Config struct {
	Jira        JiraConfig        `yaml:"jira"`
	GitHub      GitHubConfig      `yaml:"github"`
	Git         GitConfig         `yaml:"git"`
	PullRequest PullRequestConfig `yaml:"pull_request"`
	LogLevel    string            `yaml:"log_level"`
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
func DefaultConfig() *Config {
	return &Config{
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
			Template:     "# What this Pull Request does/why we need it\n\n{jira_description}\n\n{commit_messages}\n\n## What type of PR is this?\n\nfeature\n\n## Relevant links\n\n* [{jira_id}]({jira_url})\n",
		},
		LogLevel: "INFO",
	}
}

// LoadEffectiveConfig loads the configuration using the priority:
// custom path -> global path -> defaults. Repo-specific and overrides are
// intended but not implemented yet in this scaffolding.
func LoadEffectiveConfig(customPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Helper to load YAML into cfg
	load := func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Unmarshal into a temporary then merge; for scaffolding, overwrite
		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return err
		}
		mergeInto(cfg, &fileCfg)
		return nil
	}

	if customPath != "" {
		if err := load(customPath); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		return cfg, nil
	}

	globalPath := GlobalConfigPath()
	if err := load(globalPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return cfg, nil
}

// GlobalConfigPath returns ~/.config/git-autometa/config.yaml
func GlobalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to current directory
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "git-autometa", "config.yaml")
}

// RepoConfigPath returns ~/.config/git-autometa/repositories/{owner}_{repo}.yaml
func RepoConfigPath(owner, repo string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("repositories", owner+"_"+repo+".yaml")
	}
	return filepath.Join(home, ".config", "git-autometa", "repositories", owner+"_"+repo+".yaml")
}

// mergeInto merges non-zero values from src into dst.
// This is a minimal shallow merge suitable for scaffolding.
func mergeInto(dst, src *Config) {
	// Jira
	if src.Jira.ServerURL != "" {
		dst.Jira.ServerURL = src.Jira.ServerURL
	}
	if src.Jira.Email != "" {
		dst.Jira.Email = src.Jira.Email
	}

	// GitHub
	if src.GitHub.Owner != "" {
		dst.GitHub.Owner = src.GitHub.Owner
	}
	if src.GitHub.Repo != "" {
		dst.GitHub.Repo = src.GitHub.Repo
	}

	// Git
	if src.Git.BranchPattern != "" {
		dst.Git.BranchPattern = src.Git.BranchPattern
	}
	if src.Git.MaxBranchLength != 0 {
		dst.Git.MaxBranchLength = src.Git.MaxBranchLength
	}

	// PR
	if src.PullRequest.TitlePattern != "" {
		dst.PullRequest.TitlePattern = src.PullRequest.TitlePattern
	}
	dst.PullRequest.Draft = src.PullRequest.Draft || dst.PullRequest.Draft
	if src.PullRequest.BaseBranch != "" {
		dst.PullRequest.BaseBranch = src.PullRequest.BaseBranch
	}
	if src.PullRequest.Template != "" {
		dst.PullRequest.Template = src.PullRequest.Template
	}

	// Log level
	if src.LogLevel != "" {
		dst.LogLevel = src.LogLevel
	}
}
