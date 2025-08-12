# git-autometa

A Go CLI that automates JIRA-based Git workflows. Create branches and pull requests from JIRA issue information, with sensible defaults and centralized configuration.

## Features

- 🎯 **Create git branches** from JIRA issues using configurable patterns
- 🔄 **Generate pull requests** with JIRA metadata and a configurable template
- 🔀 **Handles existing branches** by auto-incrementing names (e.g., `...-2`, `...-3`)
- 🔐 **Secure credential management** using the OS keyring (single `git-autometa` service)
- ⚙️ **Centralized config** (global + per-repo overrides) under XDG config dir
- 🐙 **GitHub CLI integration** for PR creation

## Prerequisites

- Go 1.24+
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated
- Git repository
- JIRA account with API access

## Install

This requires [Go](https://go.dev/doc/install):

```bash
go install github.com/mbrt/git-autometa@latest
```

## Quick Start

1) Configure global settings and store the JIRA token in your keyring:

```bash
git-autometa config global
```

2) Create a branch for a JIRA issue:

```bash
# Specific issue
git-autometa start-work PROJ-123

# Or interactively select from your assigned issues
git-autometa start-work
```

3) Create a pull request for the current branch:

```bash
git-autometa create-pr
```

4) Check status and effective configuration:

```bash
git-autometa status
```

## Usage

### Commands

```bash
# Create and checkout branch for JIRA issue
git-autometa start-work JIRA-123 [--push]

# Create pull request from current branch
git-autometa create-pr [--base-branch <name>] [--no-draft]

# Configuration management
git-autometa config global        # Edit/show global config
git-autometa config repo          # Edit/show repo-specific overrides
git-autometa config show          # Show effective config (merged)

# Status
git-autometa status

# Persistent flags available on all commands
git-autometa -v <cmd>             # Verbose (adds connectivity checks in status)
git-autometa --owner <o> --repo <r> <cmd>  # Override repo autodetection for GitHub ops
```

### Interactive Issue Selection

Running `git-autometa start-work` without a key fetches up to 15 assigned issues from JIRA, prints a numbered list (key, summary, status, type), and prompts for a selection. If nothing is found or JIRA is unreachable, it falls back to manual entry.

### Branch Conflict Handling

If the desired branch already exists locally or remotely, the tool automatically appends an incrementing numeric suffix (e.g., `-2`, `-3`) until it finds a free name. The base branch is detected as `main` or `master`, fetched, and used to create the work branch.

## Configuration

Configuration is stored under the XDG config directory (e.g., `~/.config/git-autometa/`). The effective config is resolved in this order (last wins):

1. Repo-specific overrides: `~/.config/git-autometa/repositories/{owner}_{repo}.yaml`
2. Global config: `~/.config/git-autometa/config.yaml`
3. Built-in defaults

Layout:

```
~/.config/git-autometa/
├── config.yaml
└── repositories/
    ├── my-user_my-repo.yaml
    └── ...
```

See `example-config.yaml` for the full schema and defaults. A minimal example:

```yaml
jira:
  server_url: "https://your-company.atlassian.net"
  email: "your.email@company.com"

git:
  branch_pattern: "feature/{jira_id}-{jira_title}"
  max_branch_length: 50

pull_request:
  title_pattern: "{jira_id}: {jira_title}"
  draft: true
  base_branch: "main"
  template: |-
    # What this Pull Request does/why we need it

    {jira_description}

    {commit_messages}

    ## Relevant links
    * [{jira_id}]({jira_url})
```

### Placeholders

- `{jira_id}`: JIRA issue key (e.g., `PROJ-123`)
- `{jira_title}`: Issue title, slugified
- `{jira_type}`: Issue type (lowercased)
- `{jira_description}`: Issue description converted to Markdown
- `{jira_url}`: Link to the JIRA issue
- `{commit_messages}`: Bulleted list of commit subjects from `base..HEAD`, with common JIRA tags stripped

## Security

- JIRA API tokens are stored in the system keyring under service `git-autometa` with keys like `jira:<email>`.
- GitHub authentication is delegated to the `gh` CLI (no tokens stored here).

## Workflows

### Basic

```bash
git-autometa start-work PROJ-123
git add -A && git commit -m "Fix login validation"
git push -u origin $(git rev-parse --abbrev-ref HEAD)
git-autometa create-pr
```

### Multiple developers on the same issue

```bash
# Developer 1
git-autometa start-work PROJ-123   # feature/PROJ-123-fix-login-bug

# Developer 2 (later)
git-autometa start-work PROJ-123   # feature/PROJ-123-fix-login-bug-2
```

### Push immediately and create a ready-for-review PR

```bash
git-autometa start-work PROJ-123 --push
# ... commits ...
git-autometa create-pr --no-draft --base-branch develop
```

## Troubleshooting

- GitHub CLI not authenticated:

```bash
gh auth login
```

- Configure or update JIRA credentials:

```bash
git-autometa config global
```

- Check environment and config:

```bash
git-autometa status
```

## Development

```bash
go build .
go test ./...
```

Project layout (key parts):

```
internal/
  cli/        # Cobra commands: start-work, create-pr, config, status
  config/     # Config types, defaults, XDG paths, load/merge, save repo overrides
  git/        # Git operations (branching, commits, remotes)
  github/     # GitHub client using gh CLI
  jira/       # Jira client and Issue model
  markdown/   # Jira → Markdown converter
  secrets/    # OS keyring wrapper
```

## License

MIT License - see `LICENSE`.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `go test ./...`
5. Submit a pull request
