package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/git"
	"git-autometa/internal/secrets"
)

// Note: secrets are stored via the centralized secrets package.

var (
	configGlobalShow bool
	configRepoShow   bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage git-autometa configuration",
}

var configGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "Edit or show global configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configGlobalShow {
			return runConfigShowGlobal(cmd.OutOrStdout())
		}
		return runConfigEditGlobal(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

var configRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Edit or show repository-specific configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configRepoShow {
			return runConfigShowRepo(cmd.OutOrStdout())
		}
		return runConfigEditRepo(cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigShowEffective(cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	// global
	configCmd.AddCommand(configGlobalCmd)
	configGlobalCmd.Flags().BoolVar(&configGlobalShow, "show", false, "Show current global configuration")
	// repo
	configCmd.AddCommand(configRepoCmd)
	configRepoCmd.Flags().BoolVar(&configRepoShow, "show", false, "Show current repository configuration")
	// show
	configCmd.AddCommand(configShowCmd)
}

func loadConfig() (appconfig.Config, error) {
	paths := []string{appconfig.GlobalConfigPath()}
	owner, repo := resolveOwnerRepo()
	if owner != "" && repo != "" {
		paths = append(paths, appconfig.RepoConfigPath(owner, repo))
	}
	return appconfig.LoadEffectiveConfig(paths...)
}

func runConfigShowGlobal(out io.Writer) error {
	path := appconfig.GlobalConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "Global config not found at %s\n", path)
			return nil
		}
		return err
	}
	fmt.Fprintf(out, "Global config file: %s\n\n", path)
	_, _ = out.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Fprintln(out)
	}
	return nil
}

func runConfigShowRepo(out io.Writer) error {
	owner, repo := resolveOwnerRepo()
	if owner == "" || repo == "" {
		return errors.New("repository owner/repo not detected")
	}
	path := appconfig.RepoConfigPath(owner, repo)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "Repo config not found at %s\nShowing global config instead\n", path)
			runConfigShowGlobal(out)
			return nil
		}
		return err
	}
	fmt.Fprintf(out, "Repo config file (%s/%s): %s\n\n", owner, repo, path)
	_, _ = out.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Fprintln(out)
	}
	return nil
}

func runConfigShowEffective(out io.Writer) error {
	// Start from defaults merged with global
	paths := []string{appconfig.GlobalConfigPath()}
	// Merge repo-specific overrides if available
	owner, repo := resolveOwnerRepo()
	if owner != "" && repo != "" {
		paths = append(paths, appconfig.RepoConfigPath(owner, repo))
	}
	cfg, err := appconfig.LoadEffectiveConfig(paths...)
	if err != nil {
		return err
	}
	// Print effective config as YAML
	outYAML, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(outYAML))
	return nil
}

func runConfigEditGlobal(in io.Reader, out io.Writer) error {
	// Load current effective to use as defaults in the wizard
	cfg, err := appconfig.LoadGlobalConfig()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(in)
	fmt.Fprintf(out, "JIRA server URL [%s]: ", cfg.Jira.ServerURL)
	if s := readString(reader); s != "" {
		cfg.Jira.ServerURL = s
	}
	fmt.Fprintf(out, "JIRA email [%s]: ", cfg.Jira.Email)
	var emailInput string
	if s := readString(reader); s != "" {
		emailInput = s
		cfg.Jira.Email = emailInput
	} else {
		emailInput = cfg.Jira.Email
	}
	// Read token (not masked in basic stdin)
	fmt.Fprint(out, "JIRA API token [enter to skip storing]: ")
	tokenInput := readString(reader)

	// Write global config file
	path := appconfig.GlobalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}

	// Store token (optional)
	if tokenInput != "" && emailInput != "" {
		if err := secrets.SetJiraToken(emailInput, tokenInput); err != nil {
			return fmt.Errorf("failed saving token to keyring: %w", err)
		}
	}
	fmt.Fprintf(out, "Saved global configuration to %s\n", path)
	if tokenInput != "" {
		fmt.Fprintln(out, "Stored JIRA token in system keyring.")
	}
	return nil
}

func runConfigEditRepo(in io.Reader, out io.Writer) error {
	owner, repo := resolveOwnerRepo()
	if owner == "" || repo == "" {
		return errors.New("repository owner/repo not detected")
	}
	reader := bufio.NewReader(in)

	// Load current effective config as defaults for prompts
	baseCfg, err := loadConfig()
	if err != nil {
		return err
	}
	overrides := appconfig.Config{}
	fmt.Fprintf(out, "Branch pattern [%s]: ", baseCfg.Git.BranchPattern)
	if s := readString(reader); s != "" {
		overrides.Git.BranchPattern = s
	}
	fmt.Fprintf(out, "Max branch length [%d]: ", baseCfg.Git.MaxBranchLength)
	if n, ok := readInt(reader); ok {
		overrides.Git.MaxBranchLength = n
	}
	fmt.Fprintf(out, "PR title pattern [%s]: ", baseCfg.PullRequest.TitlePattern)
	if s := readString(reader); s != "" {
		overrides.PullRequest.TitlePattern = s
	}
	fmt.Fprintf(out, "PR base branch [%s]: ", baseCfg.PullRequest.BaseBranch)
	if s := readString(reader); s != "" {
		overrides.PullRequest.BaseBranch = s
	}
	fmt.Fprintf(out, "PR draft by default (true/false) [%t]: ", baseCfg.PullRequest.Draft)
	if v, _ := reader.ReadString('\n'); strings.TrimSpace(v) != "" {
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "true", "t", "yes", "y":
			overrides.PullRequest.Draft = true
		case "false", "f", "no", "n":
			// keep false (zero) to avoid overriding config with explicit false
		}
	}

	// Persist repo-specific config using dedicated config package function
	path, err := appconfig.SaveRepoConfig(owner, repo, overrides)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved repository configuration for %s/%s to %q\n", owner, repo, path)
	return nil
}

func resolveOwnerRepo() (string, string) {
	// Flags override autodetection
	owner := repoOwner
	repo := repoName
	if owner != "" && repo != "" {
		return owner, repo
	}
	// Try git remote
	gitUtils := git.New()
	remoteURL, err := gitUtils.GetRemoteURL("origin")
	if err == nil {
		if o, r, ok := parseGitHubOwnerRepo(remoteURL); ok {
			if owner == "" {
				owner = o
			}
			if repo == "" {
				repo = r
			}
		}
	}
	return owner, repo
}

func parseGitHubOwnerRepo(remote string) (string, string, bool) {
	s := strings.TrimSuffix(remote, ".git")
	// Normalize ssh scp-like: git@github.com:owner/repo -> ssh://git@github.com/owner/repo
	if strings.HasPrefix(s, "git@github.com:") {
		s = strings.Replace(s, ":", "/", 1)
		s = "ssh://" + s
	}
	// Find host separator
	hostIdx := strings.Index(s, "github.com")
	if hostIdx < 0 {
		return "", "", false
	}
	after := s[hostIdx+len("github.com"):]
	after = strings.TrimPrefix(after, "/")
	parts := strings.Split(after, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner := parts[0]
	repo := parts[1]
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}

// readString reads a line, trims whitespace and the trailing newline.
func readString(r *bufio.Reader) string {
	v, _ := r.ReadString('\n')
	return strings.TrimSpace(v)
}

// readInt reads a line and parses a trimmed integer. Returns (value, true) on success.
func readInt(r *bufio.Reader) (int, bool) {
	s := readString(r)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
