package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

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

// LoadEffectiveConfig loads the configuration using the priority:
// custom path -> global path -> defaults. Repo-specific and overrides are
// intended but not implemented yet in this scaffolding.
func LoadEffectiveConfig(customPath string) (Config, error) {
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
		mergeInto(&cfg, &fileCfg)
		return nil
	}

	if customPath != "" {
		if err := load(customPath); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return Config{}, err
			}
		}
		return cfg, nil
	}

	globalPath := GlobalConfigPath()
	if err := load(globalPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Config{}, err
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

// mergeInto merges non-zero values from src into dst.
// This is a minimal shallow merge suitable for scaffolding.
func mergeInto(dst, src *Config) {
	if dst == nil || src == nil {
		return
	}
	mergeStruct(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
}

// mergeStruct copies non-zero fields from src into dst. It recurses into nested structs.
// For booleans, only true overrides (keeps previous behavior of only setting when true).
func mergeStruct(dst, src reflect.Value) {
	if dst.Kind() != reflect.Struct || src.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < dst.NumField(); i++ {
		dstField := dst.Field(i)
		srcField := src.Field(i)

		if !dstField.CanSet() {
			continue
		}

		switch dstField.Kind() {
		case reflect.Struct:
			mergeStruct(dstField, srcField)
		default:
			// Only overwrite when the source field is non-zero
			if !srcField.IsZero() {
				dstField.Set(srcField)
			}
		}
	}
}
