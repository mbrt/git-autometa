# git-autometa

A Python tool for automating JIRA-based git workflows. Automatically create branches and pull requests based on JIRA issue information.

## Features

- 🎯 **Create git branches** from JIRA issues with configurable naming patterns
- 🔄 **Generate pull requests** with JIRA metadata and custom templates
- 🔀 **Multiple branch support** - handles local/remote conflicts intelligently
- 🔐 **Secure credential management** using keyring for JIRA API tokens
- 🎨 **Rich CLI interface** with colored output and progress indicators
- ⚙️ **Highly configurable** with YAML configuration files
- 🐙 **GitHub CLI integration** for seamless PR creation

## Prerequisites

- Python 3.8+
- [uv](https://github.com/astral-sh/uv) package manager
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated
- Git repository
- JIRA account with API access

## Installation

```bash
# Clone and install with pipx
git clone https://github.com/mbrt/git-autometa.git
pipx install .
```

## Quick Start

1. **Configure git-autometa globally:**
   ```bash
   git-autometa config global
   ```
   This will prompt you for:
   - JIRA server URL
   - JIRA email
   - JIRA API token (stored securely in keyring)

2. **Create a branch for a JIRA issue:**
   ```bash
   git-autometa start-work PROJ-123
   ```

3. **Create a pull request from current branch:**
   ```bash
   git-autometa create-pr
   ```

4. **Check status:**
   ```bash
   git-autometa status
   ```

## Usage

### Main Commands

```bash
# Create and checkout branch for JIRA issue
git-autometa start-work JIRA-123

# Push branch to remote after creation
git-autometa start-work JIRA-123 --push

# Create pull request from current branch
git-autometa create-pr

# Create non-draft PR
git-autometa create-pr --no-draft

# Use custom base branch for PR
git-autometa create-pr --base-branch develop
```

### Multiple Branch Support

The `start-work` command intelligently handles branch conflicts:

When a branch already exists (locally, remotely, or both), you'll be prompted to choose:
1. **Switch to existing branch** - Checkout the existing branch
2. **Create alternative branch** - Auto-generate a new name (e.g., `feature/PROJ-123-fix-bug-2`)

This enables multiple developers to work on the same JIRA issue with separate branches.

**Example scenarios:**
```bash
# First time - creates feature/PROJ-123-fix-bug
git-autometa start-work PROJ-123

# Later, same issue - prompts for action
git-autometa start-work PROJ-123
# Output: Branch 'feature/PROJ-123-fix-bug' already exists locally
# Choose: 1) Switch to existing branch  2) Create new branch with alternative name
# Choosing 2 creates: feature/PROJ-123-fix-bug-2
```

### Configuration Commands

```bash
# Configure global settings (applies to all repositories)
git-autometa config global

# Configure repository-specific settings (overrides global)
git-autometa config repo

# Show current configuration and sources
git-autometa config show

# Show current status and configuration
git-autometa status

# Enable verbose logging
git-autometa -v start-work JIRA-123
```

## Configuration

git-autometa uses a centralized configuration system that keeps configuration files outside of your repositories:

### Configuration Hierarchy

1. **Command line arguments** (highest priority)
2. **Repository-specific config** - `~/.config/git-autometa/repositories/{owner}_{repo}.yaml`
3. **Global config** - `~/.config/git-autometa/config.yaml`
4. **Built-in defaults** (lowest priority)

### Configuration Structure

```
~/.config/git-autometa/
├── config.yaml                    # Global defaults
└── repositories/
    ├── my-user_my-repo.yaml       # Repo-specific config
    ├── company_project-api.yaml   # Another repo config
    └── ...
```

### Global Configuration Example

```yaml
# ~/.config/git-autometa/config.yaml
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
  template_path: "templates/pr_template.md"

log_level: "INFO"
```

### Repository-Specific Configuration Example

```yaml
# ~/.config/git-autometa/repositories/my-user_my-repo.yaml
# Only specify values you want to override from global config
jira:
  server_url: "https://different-jira.atlassian.net"  # Override global

git:
  branch_pattern: "bugfix/{jira_id}"  # Different pattern for this repo
  max_branch_length: 40

pull_request:
  base_branch: "develop"  # This repo uses develop instead of main
```

### Custom Configuration Path

You can also specify a custom configuration file path:

```bash
git-autometa --config /path/to/custom-config.yaml create PROJ-123
```

### Branch Naming Patterns

Available placeholders:
- `{jira_id}` - JIRA issue key (e.g., PROJ-123)
- `{jira_title}` - JIRA issue title (slugified)
- `{jira_type}` - JIRA issue type (e.g., bug, feature)

Examples:
- `feature/{jira_id}-{jira_title}` → `feature/PROJ-123-fix-login-bug`
- `{jira_type}/{jira_id}` → `bug/PROJ-123`
- `{jira_id}` → `PROJ-123`

### PR Templates

Create custom PR templates with JIRA metadata auto-population:

```markdown
# What this Pull Request does/why we need it

{jira_description}

**JIRA Issue:** [{jira_id}]({jira_url})

## Type of change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Links
- JIRA Issue: [{jira_id}]({jira_url})
```

Available template placeholders:
- `{jira_id}` - Issue key
- `{jira_title}` - Issue title
- `{jira_description}` - Issue description
- `{jira_url}` - Direct link to JIRA issue
- `{jira_type}` - Issue type

## Security

- **JIRA API tokens** are stored securely using the system keyring
- **GitHub authentication** uses the GitHub CLI's existing auth
- **No secrets** are stored in configuration files

## Workflow Examples

### Basic Workflow
```bash
# 1. Create and checkout branch
git-autometa start-work PROJ-123

# 2. Make your changes
git add .
git commit -m "Fix login validation"

# 3. Push changes
git push -u origin feature/PROJ-123-fix-login-bug

# 4. Create pull request
git-autometa create-pr
```

### Multiple Developers on Same Issue
```bash
# Developer 1
git-autometa start-work PROJ-123
# Creates: feature/PROJ-123-fix-login-bug

# Developer 2 (later)
git-autometa start-work PROJ-123
# Prompts: Branch exists remotely. Choose:
# 1) Switch to existing branch
# 2) Create alternative branch
# Choosing 2 creates: feature/PROJ-123-fix-login-bug-2
```

### Push and Create PR Workflow
```bash
# 1. Create branch with immediate push
git-autometa start-work PROJ-123 --push

# 2. Make changes
git add .
git commit -m "Fix login validation"
git push

# 3. Create ready-for-review PR
git-autometa create-pr --no-draft --base-branch develop
```

## Troubleshooting

### Common Issues

1. **GitHub CLI not authenticated:**
   ```bash
   gh auth login
   ```

2. **JIRA API token issues:**
   ```bash
   git-autometa config global  # Re-configure JIRA credentials
   ```

3. **Template not found:**
   - Check `pull_request.template_path` in config
   - Ensure template file exists

4. **Branch already exists:**
   - Tool will checkout existing branch instead of creating new one
   - Use different branch naming pattern if needed

### Debug Mode

Enable verbose logging for troubleshooting:
```bash
git-autometa -v start-work PROJ-123
```

## Development

### Setting up development environment

```bash
# Install development dependencies
uv sync --dev

# Run tests
pytest

# Format code
black src/

# Type checking
mypy src/
```

### Project Structure

```
git-autometa/
├── src/git_autometa/
│   ├── __init__.py
│   ├── main.py          # CLI entry point
│   ├── config.py        # Configuration management
│   ├── jira_client.py   # JIRA API client
│   ├── github_client.py # GitHub CLI wrapper
│   └── git_utils.py     # Git operations
├── templates/
│   └── pr_template.md   # Default PR template
├── config.yaml          # Default configuration
├── pyproject.toml       # Project configuration
└── README.md
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request
