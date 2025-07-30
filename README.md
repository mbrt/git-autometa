# git-autometa

A Python tool for automating JIRA-based git workflows. Automatically create branches and pull requests based on JIRA issue information.

## Features

- 🎯 **Create git branches** from JIRA issues with configurable naming patterns
- 🔄 **Generate pull requests** with JIRA metadata and custom templates
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
# Clone the repository
git clone <repository-url>
cd git-autometa

# Install with uv
uv sync
uv pip install -e .
```

## Quick Start

1. **Configure git-autometa:**
   ```bash
   git-autometa config
   ```
   This will prompt you for:
   - JIRA server URL
   - JIRA email
   - JIRA API token (stored securely in keyring)

2. **Create a branch and PR for a JIRA issue:**
   ```bash
   git-autometa create PROJ-123
   ```

3. **Check status:**
   ```bash
   git-autometa status
   ```

## Usage

### Main Commands

```bash
# Create branch and PR for JIRA issue
git-autometa create JIRA-123

# Create only branch (skip PR)
git-autometa create JIRA-123 --branch-only

# Create only PR (from current branch)
git-autometa create JIRA-123 --pr-only

# Push branch to remote after creation
git-autometa create JIRA-123 --push

# Create non-draft PR
git-autometa create JIRA-123 --no-draft

# Use custom base branch
git-autometa create JIRA-123 --base-branch develop
```

### Configuration Commands

```bash
# Configure credentials and settings
git-autometa config

# Show current status and configuration
git-autometa status

# Enable verbose logging
git-autometa -v create JIRA-123
```

## Configuration

git-autometa uses YAML configuration files. The tool looks for `config.yaml` in:
1. Current directory
2. Git repository root
3. Package default location

### Example Configuration

```yaml
# config.yaml
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
# 1. Create branch and PR
git-autometa create PROJ-123

# 2. Make your changes
git add .
git commit -m "Fix login validation"

# 3. Push changes (branch already exists remotely)
git push
```

### Branch-only Workflow
```bash
# 1. Create branch only
git-autometa create PROJ-123 --branch-only

# 2. Make changes and push
git add .
git commit -m "Fix login validation"
git push -u origin feature/PROJ-123-fix-login-bug

# 3. Create PR later
git-autometa create PROJ-123 --pr-only
```

### Custom Workflow
```bash
# Create ready-for-review PR on develop branch
git-autometa create PROJ-123 \
  --base-branch develop \
  --no-draft \
  --push
```

## Troubleshooting

### Common Issues

1. **GitHub CLI not authenticated:**
   ```bash
   gh auth login
   ```

2. **JIRA API token issues:**
   ```bash
   git-autometa config  # Re-configure JIRA credentials
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
git-autometa -v create PROJ-123
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
