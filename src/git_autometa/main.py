"""
Main CLI for git-autometa
"""

import click
import logging
import sys
from pathlib import Path
from rich.console import Console
from rich.logging import RichHandler

from .config import Config
from .jira_client import JiraClient, JiraIssue
from .github_client import GitHubClient
from .git_utils import GitUtils

console = Console()


def setup_logging(level: str = "INFO"):
    """Setup logging with rich handler"""
    logging.basicConfig(
        level=getattr(logging, level.upper()),
        format="%(message)s",
        datefmt="[%X]",
        handlers=[RichHandler(console=console, rich_tracebacks=True)]
    )


def format_branch_name(pattern: str, issue: JiraIssue, max_length: int = 50) -> str:
    """
    Format branch name using pattern and issue data

    Args:
        pattern: Branch naming pattern
        issue: JIRA issue
        max_length: Maximum branch name length

    Returns:
        Formatted branch name
    """
    # Calculate remaining length after jira_id
    jira_id_length = len(issue.key)
    remaining_length = max_length - jira_id_length - \
        10  # Reserve space for pattern and separators

    branch_name = pattern.format(
        jira_id=issue.key,
        jira_title=issue.slugify_title(max(remaining_length, 10)),
        jira_type=issue.issue_type
    )

    # Truncate if still too long
    if len(branch_name) > max_length:
        branch_name = branch_name[:max_length]

    return branch_name


def load_pr_template(template_path: str, issue: JiraIssue) -> str:
    """
    Load and format PR template

    Args:
        template_path: Path to template file
        issue: JIRA issue

    Returns:
        Formatted template content
    """
    try:
        # Try relative to current directory first
        path = Path(template_path)
        if not path.exists():
            # Try relative to git root
            git_utils = GitUtils()
            git_root = Path(git_utils.repo.working_dir)
            path = git_root / template_path

        if not path.exists():
            # Try package default
            package_dir = Path(__file__).parent.parent.parent
            path = package_dir / template_path

        if path.exists():
            content = path.read_text()
            return content.format(
                jira_id=issue.key,
                jira_title=issue.summary,
                jira_description=issue.description or "See JIRA issue for details.",
                jira_url=issue.url,
                jira_type=issue.issue_type
            )
        else:
            console.print(
                f"[yellow]Warning: Template not found at {template_path}[/yellow]")
            return f"Resolves {issue.key}: {issue.summary}\n\n{issue.url}"

    except Exception as e:
        console.print(
            f"[yellow]Warning: Failed to load template: {e}[/yellow]")
        return f"Resolves {issue.key}: {issue.summary}\n\n{issue.url}"


@click.group()
@click.option('--config', '-c', help='Configuration file path')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose logging')
@click.pass_context
def cli(ctx, config, verbose):
    """git-autometa: Automate JIRA-based git workflows"""
    ctx.ensure_object(dict)

    # Setup logging
    log_level = "DEBUG" if verbose else "INFO"
    setup_logging(log_level)

    # Load configuration
    try:
        ctx.obj['config'] = Config(config)
        if verbose:
            console.print(
                f"[dim]{ctx.obj['config'].config_path_info}[/dim]")
    except Exception as e:
        console.print(f"[red]Error loading configuration: {e}[/red]")
        sys.exit(1)


@cli.command()
@click.argument('jira_issue')
@click.option('--branch-only', is_flag=True, help='Only create branch, skip PR')
@click.option('--pr-only', is_flag=True, help='Only create PR, skip branch creation')
@click.option('--base-branch', help='Base branch for PR (overrides config)')
@click.option('--no-draft', is_flag=True, help='Create PR as ready for review')
@click.option('--push', is_flag=True, help='Push branch to remote')
@click.pass_context
def create(ctx, jira_issue, branch_only, pr_only, base_branch, no_draft, push):
    """Create branch and/or PR for JIRA issue"""
    config = ctx.obj['config']

    try:
        # Initialize clients
        console.print("[bold blue]Initializing clients...[/bold blue]")

        # JIRA client
        if not config.jira_server_url or not config.jira_email:
            console.print(
                "[red]JIRA configuration missing. Run 'git-autometa config global' first.[/red]")
            sys.exit(1)

        jira_client = JiraClient(config.jira_server_url, config.jira_email)

        # GitHub client (only if creating PR)
        github_client = None
        if not branch_only:
            github_client = GitHubClient()

        # Git utils
        git_utils = GitUtils()

        # Fetch JIRA issue
        console.print(
            f"[bold blue]Fetching JIRA issue: {jira_issue}[/bold blue]")
        issue = jira_client.get_issue(jira_issue)
        console.print(f"[green]✓[/green] Found issue: {issue.summary}")

        branch_name = ""

        # Create branch if requested
        if not pr_only:
            console.print("[bold blue]Creating git branch...[/bold blue]")

            # Generate branch name
            branch_name = format_branch_name(
                config.branch_pattern,
                issue,
                config.max_branch_length
            )

            console.print(f"Branch name: {branch_name}")

            # Check if branch already exists
            if git_utils.branch_exists(branch_name):
                console.print(
                    f"[yellow]Branch '{branch_name}' already exists locally[/yellow]")
                git_utils.checkout_branch(branch_name)
            else:
                # Create and checkout branch
                git_utils.create_branch(branch_name, checkout=True)
                console.print(
                    f"[green]✓[/green] Created and checked out branch: {branch_name}")

            # Push branch if requested
            if push:
                console.print(
                    "[bold blue]Pushing branch to remote...[/bold blue]")
                git_utils.push_branch(branch_name)
                console.print(f"[green]✓[/green] Pushed branch to remote")

        # Create PR if requested
        if not branch_only:
            console.print("[bold blue]Creating pull request...[/bold blue]")

            # Get current branch if not creating one
            if not branch_name:
                branch_name = git_utils.get_current_branch()

            # Generate PR title
            pr_title = config.pr_title_pattern.format(
                jira_id=issue.key,
                jira_title=issue.summary,
                jira_type=issue.issue_type
            )

            # Load PR template
            pr_body = load_pr_template(config.pr_template_path, issue)

            # Determine base branch
            base = base_branch or config.pr_base_branch
            if not base and github_client:
                base = github_client.get_default_branch()
            elif not base:
                base = "main"  # Fallback

            # Create PR
            if github_client:
                draft = config.pr_draft and not no_draft
                pr_url = github_client.create_pull_request(
                    title=pr_title,
                    body=pr_body,
                    head=branch_name,
                    base=base,
                    draft=draft
                )
            else:
                raise ValueError("GitHub client not initialized")

            console.print(f"[green]✓[/green] Created pull request: {pr_url}")

        console.print(
            "[bold green]✅ Workflow completed successfully![/bold green]")

    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        sys.exit(1)


@cli.group()
@click.pass_context
def config_cmd(ctx):
    """Configure git-autometa settings"""
    pass


@config_cmd.command('global')
@click.pass_context
def config_global(ctx):
    """Configure global git-autometa settings"""
    config = ctx.obj['config']

    console.print("[bold blue]Configuring global git-autometa settings...[/bold blue]")

    # JIRA configuration
    console.print("\n[bold]JIRA Configuration[/bold]")

    current_server = config.jira_server_url
    server_url = click.prompt(
        f"JIRA Server URL",
        default=current_server if current_server else "https://your-company.atlassian.net"
    )
    config.set_global('jira.server_url', server_url)

    current_email = config.jira_email
    email = click.prompt(
        f"JIRA Email",
        default=current_email if current_email else ""
    )
    config.set_global('jira.email', email)

    # Get JIRA API token
    api_token = click.prompt("JIRA API Token", hide_input=True)
    JiraClient.store_api_token(email, api_token)

    # Test JIRA connection
    try:
        jira_client = JiraClient(server_url, email)
        if jira_client.test_connection():
            console.print("[green]✓[/green] JIRA connection successful")
        else:
            console.print("[red]✗[/red] JIRA connection failed")
    except Exception as e:
        console.print(f"[red]✗[/red] JIRA connection failed: {e}")

    # Save configuration
    if config.custom_config_path:
        config.save_custom()
    else:
        config.save_global()
    console.print("[green]✓[/green] Global configuration saved")


@config_cmd.command('repo')
@click.pass_context
def config_repo(ctx):
    """Configure repository-specific git-autometa settings"""
    config = ctx.obj['config']

    # Check if we're in a git repository
    repo_id = config.get_current_repo_id()
    if not repo_id:
        console.print("[red]Error: Not in a git repository with GitHub remote[/red]")
        sys.exit(1)

    console.print(f"[bold blue]Configuring settings for repository: {repo_id.replace('_', '/')}[/bold blue]")

    # Show current effective configuration
    console.print("\n[dim]Current effective configuration:[/dim]")
    console.print(f"[dim]JIRA Server: {config.jira_server_url}[/dim]")
    console.print(f"[dim]JIRA Email: {config.jira_email}[/dim]")
    console.print(f"[dim]Branch Pattern: {config.branch_pattern}[/dim]")
    console.print(f"[dim]PR Title Pattern: {config.pr_title_pattern}[/dim]")

    console.print("\n[bold]Repository-specific overrides[/bold]")
    console.print("[dim]Leave empty to use global defaults[/dim]")

    # JIRA configuration overrides
    console.print("\n[bold]JIRA Configuration[/bold]")
    
    server_url = click.prompt(
        f"JIRA Server URL (current: {config.jira_server_url})",
        default="",
        show_default=False
    )
    if server_url:
        config.set_repo('jira.server_url', server_url)

    email = click.prompt(
        f"JIRA Email (current: {config.jira_email})",
        default="",
        show_default=False
    )
    if email:
        config.set_repo('jira.email', email)

    # Git configuration overrides
    console.print("\n[bold]Git Configuration[/bold]")
    
    branch_pattern = click.prompt(
        f"Branch Pattern (current: {config.branch_pattern})",
        default="",
        show_default=False
    )
    if branch_pattern:
        config.set_repo('git.branch_pattern', branch_pattern)

    max_branch_length = click.prompt(
        f"Max Branch Length (current: {config.max_branch_length})",
        type=int,
        default=0,
        show_default=False
    )
    if max_branch_length > 0:
        config.set_repo('git.max_branch_length', max_branch_length)

    # Pull Request configuration overrides
    console.print("\n[bold]Pull Request Configuration[/bold]")
    
    pr_title_pattern = click.prompt(
        f"PR Title Pattern (current: {config.pr_title_pattern})",
        default="",
        show_default=False
    )
    if pr_title_pattern:
        config.set_repo('pull_request.title_pattern', pr_title_pattern)

    pr_draft = click.prompt(
        f"Create Draft PRs (current: {config.pr_draft})",
        type=bool,
        default=None,
        show_default=False
    )
    if pr_draft is not None:
        config.set_repo('pull_request.draft', pr_draft)

    pr_base_branch = click.prompt(
        f"PR Base Branch (current: {config.pr_base_branch})",
        default="",
        show_default=False
    )
    if pr_base_branch:
        config.set_repo('pull_request.base_branch', pr_base_branch)

    # Save repository configuration
    config.save_repo()
    console.print("[green]✓[/green] Repository configuration saved")


@config_cmd.command('show')
@click.pass_context
def config_show(ctx):
    """Show current configuration"""
    config = ctx.obj['config']

    console.print("[bold blue]Current git-autometa Configuration[/bold blue]")

    # Show configuration sources
    console.print(f"\n[bold]Configuration Sources[/bold]")
    console.print(config.config_path_info)

    repo_id = config.get_current_repo_id()
    if repo_id:
        console.print(f"\nCurrent repository: {repo_id.replace('_', '/')}")

    # Show effective configuration
    console.print(f"\n[bold]Effective Configuration[/bold]")
    
    console.print("\n[bold]JIRA[/bold]")
    console.print(f"Server URL: {config.jira_server_url or '[dim]Not set[/dim]'}")
    console.print(f"Email: {config.jira_email or '[dim]Not set[/dim]'}")

    console.print("\n[bold]Git[/bold]")
    console.print(f"Branch Pattern: {config.branch_pattern}")
    console.print(f"Max Branch Length: {config.max_branch_length}")

    console.print("\n[bold]Pull Request[/bold]")
    console.print(f"Title Pattern: {config.pr_title_pattern}")
    console.print(f"Draft: {config.pr_draft}")
    console.print(f"Base Branch: {config.pr_base_branch}")
    console.print(f"Template Path: {config.pr_template_path}")

    console.print("\n[bold]Logging[/bold]")
    console.print(f"Log Level: {config.log_level}")


@cli.command()
@click.pass_context
def status(ctx):
    """Show git-autometa status and configuration"""
    config = ctx.obj['config']

    console.print("[bold blue]git-autometa Status[/bold blue]")

    # Configuration status
    console.print(f"\n[bold]Configuration[/bold]")
    console.print(config.config_path_info)
    console.print(
        f"JIRA Server: {config.jira_server_url or '[red]Not configured[/red]'}")
    console.print(
        f"JIRA Email: {config.jira_email or '[red]Not configured[/red]'}")

    # Git repository status
    try:
        git_utils = GitUtils()
        console.print(f"\n[bold]Git Repository[/bold]")
        console.print(f"Repository: {git_utils.repo.working_dir}")
        console.print(f"Current branch: {git_utils.get_current_branch()}")

        changes = git_utils.get_uncommitted_changes()
        if changes:
            console.print(f"Uncommitted changes: {len(changes)} files")
        else:
            console.print("Working directory: [green]clean[/green]")

    except Exception as e:
        console.print(f"\n[red]Git repository error: {e}[/red]")

    # Test connections
    console.print(f"\n[bold]Service Connections[/bold]")

    # Test JIRA
    if config.jira_server_url and config.jira_email:
        try:
            jira_client = JiraClient(config.jira_server_url, config.jira_email)
            if jira_client.test_connection():
                console.print("JIRA: [green]✓ Connected[/green]")
            else:
                console.print("JIRA: [red]✗ Connection failed[/red]")
        except Exception as e:
            console.print(f"JIRA: [red]✗ {e}[/red]")
    else:
        console.print("JIRA: [yellow]Not configured[/yellow]")

    # Test GitHub CLI
    try:
        github_client = GitHubClient()
        if github_client.test_connection():
            console.print("GitHub: [green]✓ Connected[/green]")
        else:
            console.print("GitHub: [red]✗ Connection failed[/red]")
    except Exception as e:
        console.print(f"GitHub: [red]✗ {e}[/red]")


if __name__ == '__main__':
    cli()
