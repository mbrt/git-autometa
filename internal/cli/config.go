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
	keyring "github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"

	appconfig "git-autometa/internal/config"
	"git-autometa/internal/git"
)

const jiraKeyringService = "git-autometa-jira"

var (
	configGlobalShow bool
	configRepoShow   bool
	repoOwner        string
	repoName         string
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
	configRepoCmd.Flags().StringVar(&repoOwner, "owner", "", "Repository owner (defaults to current git remote)")
	configRepoCmd.Flags().StringVar(&repoName, "repo", "", "Repository name (defaults to current git remote)")
	// show
	configCmd.AddCommand(configShowCmd)
}

func runConfigShowGlobal(out io.Writer) error {
	// Determine path precedence: explicit --config, otherwise global path
	path := cfgPath
	if strings.TrimSpace(path) == "" {
		path = appconfig.GlobalConfigPath()
	}
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
		fmt.Fprintln(out, "Repository owner/repo not detected. Use --owner and --repo flags or set in config.")
		return nil
	}
	path := appconfig.RepoConfigPath(owner, repo)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
	paths := []string{cfgPath}
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
	cfg, err := appconfig.LoadEffectiveConfig(cfgPath)
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
		if err := keyring.Set(jiraKeyringService, emailInput, tokenInput); err != nil {
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
	reader := bufio.NewReader(in)
	if owner == "" {
		fmt.Fprint(out, "Repository owner (e.g., my-user): ")
		if s := readString(reader); s != "" {
			owner = s
		}
	}
	if repo == "" {
		fmt.Fprint(out, "Repository name (e.g., my-repo): ")
		if s := readString(reader); s != "" {
			repo = s
		}
	}
	if owner == "" || repo == "" {
		return errors.New("owner/repo not provided")
	}

	// Load current effective config as defaults for prompts
	baseCfg, err := appconfig.LoadEffectiveConfig(cfgPath)
	if err != nil {
		return err
	}
	// Start with empty overrides and fill only provided fields
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

	// Persist repo-specific config
	path := appconfig.RepoConfigPath(owner, repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(overrides)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved repository configuration for %s/%s to %s\n", owner, repo, path)
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

// func readString2(dst *string, out io.Writer, r *bufio.Reader, prompt string) {
// 	fmt.Fprintf(out, "%s [%s]: ", prompt, *dst)
// 	v, _ := r.ReadString('\n')
// 	if v = strings.TrimSpace(v); v != "" {
// 		*dst = v
// 	}
// }

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
