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


def prompt_with_default(text, current_value, **kwargs):
    """Helper to prompt with a default value, allowing empty input to keep the current value."""
    # The `default` parameter to `click.prompt` is what's returned when the user just presses Enter.
    # By setting it to the current value, we achieve the desired behavior.
    # `show_default=True` (the default) would show `[current_value]` after the prompt.
    # We create a custom prompt text to make it more explicit.
    prompt_text = f"{text} (current: {current_value})"
    return click.prompt(prompt_text, default=current_value, show_default=False, **kwargs)


def select_jira_issue_interactively(jira_client: JiraClient) -> str:
    """
    Display an interactive list of assigned JIRA issues and return the selected issue key.
    
    Args:
        jira_client: Initialized JIRA client
        
    Returns:
        Selected JIRA issue key
        
    Raises:
        click.Abort: If user cancels selection
    """
    console.print("[bold blue]Fetching your assigned issues...[/bold blue]")
    
    try:
        issues = jira_client.search_my_issues()
    except Exception as e:
        console.print(f"[red]Error fetching issues: {e}[/red]")
        console.print("[yellow]Falling back to manual issue entry...[/yellow]")
        return click.prompt("Enter JIRA issue key")
    
    if not issues:
        console.print("[yellow]No issues found assigned to you.[/yellow]")
        console.print("[yellow]Falling back to manual issue entry...[/yellow]")
        return click.prompt("Enter JIRA issue key")
    
    console.print(f"\n[bold]Found {len(issues)} assigned issues:[/bold]")
    console.print()
    
    # Display numbered list of issues
    for i, issue in enumerate(issues, 1):
        # Truncate summary if too long
        summary = issue.summary
        if len(summary) > 60:
            summary = summary[:57] + "..."
        
        status_color = "green" if issue.status.lower() in ["in progress", "doing"] else "blue"
        console.print(f"[dim]{i:2}.[/dim] [bold]{issue.key}[/bold]: {summary}")
        console.print(f"    [dim]Status:[/dim] [{status_color}]{issue.status}[/{status_color}] [dim]Type:[/dim] {issue.issue_type.title()}")
        console.print()
    
    console.print("[dim]0.[/dim] [italic]Cancel[/italic]")
    console.print()
    
    # Get user selection
    while True:
        try:
            choice = click.prompt(
                "Select an issue", 
                type=int,
                show_default=False
            )
            
            if choice == 0:
                console.print("[yellow]Cancelled[/yellow]")
                raise click.Abort()
            elif 1 <= choice <= len(issues):
                selected_issue = issues[choice - 1]
                console.print(f"[green]Selected:[/green] {selected_issue.key} - {selected_issue.summary}")
                return selected_issue.key
            else:
                console.print(f"[red]Invalid choice. Please select a number between 0 and {len(issues)}[/red]")
        except click.Abort:
            raise
        except (ValueError, click.BadParameter):
            console.print(f"[red]Invalid input. Please enter a number between 0 and {len(issues)}[/red]")


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


@cli.command("start-work")
@click.argument('jira_issue', required=False)
@click.option('--push', is_flag=True, help='Push branch to remote after creation')
@click.pass_context
def start_work(ctx, jira_issue, push):
    """Creates and checks out a new branch for a JIRA issue.
    
    If no JIRA issue is provided, shows an interactive list of your assigned issues.
    """
    config = ctx.obj['config']
    try:
        console.print("[bold blue]Initializing clients...[/bold blue]")
        if not config.jira_server_url or not config.jira_email:
            console.print(
                "[red]JIRA configuration missing. Run 'git-autometa config global' first.[/red]")
            sys.exit(1)
        jira_client = JiraClient(config.jira_server_url, config.jira_email)
        git_utils = GitUtils()

        # If no issue provided, show interactive selection
        if not jira_issue:
            try:
                jira_issue = select_jira_issue_interactively(jira_client)
            except click.Abort:
                console.print("[yellow]Operation cancelled[/yellow]")
                sys.exit(0)

        console.print(
            f"[bold blue]Fetching JIRA issue: {jira_issue}[/bold blue]")
        issue = jira_client.get_issue(jira_issue)
        console.print(f"[green]✓[/green] Found issue: {issue.summary}")

        console.print("[bold blue]Creating git branch...[/bold blue]")
        base_branch_name = format_branch_name(
            config.branch_pattern, issue, config.max_branch_length)
        console.print(f"Desired branch name: {base_branch_name}")

        console.print("[bold blue]Updating main branch...[/bold blue]")
        # Use the enhanced branch preparation that handles conflicts
        final_branch_name = git_utils.prepare_work_branch(base_branch_name)
        
        if final_branch_name == base_branch_name:
            console.print(
                f"[green]✓[/green] Created and checked out branch: {final_branch_name}")
        else:
            console.print(
                f"[green]✓[/green] Using branch: {final_branch_name}")

        if push:
            console.print(
                "[bold blue]Pushing branch to remote...[/bold blue]")
            git_utils.push_branch(final_branch_name)
            console.print("[green]✓[/green] Pushed branch to remote")

        console.print(
            "[bold green]✅ Branch is ready. Happy coding![/bold green]")
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        sys.exit(1)


@cli.command("create-pr")
@click.option('--base-branch', help='Base branch for PR (overrides config)')
@click.option('--no-draft', is_flag=True, help='Create PR as ready for review')
@click.pass_context
def create_pr(ctx, base_branch, no_draft):
    """Creates a pull request for the current branch."""
    config = ctx.obj['config']
    try:
        console.print("[bold blue]Initializing clients...[/bold blue]")
        if not config.jira_server_url or not config.jira_email:
            console.print(
                "[red]JIRA configuration missing. Run 'git-autometa config global' first.[/red]")
            sys.exit(1)
        jira_client = JiraClient(config.jira_server_url, config.jira_email)
        github_client = GitHubClient()
        git_utils = GitUtils()

        branch_name = git_utils.get_current_branch()
        console.print(f"Current branch: {branch_name}")

        # Extract JIRA issue key from branch name
        # This is a simple implementation, might need to be more robust
        jira_issue_key = branch_name.split('/')[1].split('-')[0] + '-' + branch_name.split('/')[1].split('-')[1]

        console.print(
            f"[bold blue]Fetching JIRA issue: {jira_issue_key}[/bold blue]")
        issue = jira_client.get_issue(jira_issue_key)
        console.print(f"[green]✓[/green] Found issue: {issue.summary}")

        console.print("[bold blue]Creating pull request...[/bold blue]")
        pr_title = config.pr_title_pattern.format(
            jira_id=issue.key,
            jira_title=issue.summary,
            jira_type=issue.issue_type
        )
        
        base = base_branch or config.pr_base_branch or github_client.get_default_branch()
        
        # Get commit messages for PR
        commit_messages = git_utils.get_commit_messages_for_pr(base)
        
        pr_body = config.pr_template.format(
            jira_id=issue.key,
            jira_title=issue.summary,
            jira_description=issue.description_markdown or "See JIRA issue for details.",
            jira_url=issue.url,
            jira_type=issue.issue_type,
            commit_messages=commit_messages
        )
        draft = config.pr_draft and not no_draft

        # Ensure the branch is pushed before creating the PR
        if not git_utils.is_branch_pushed(branch_name):
            console.print("[bold blue]Pushing branch to remote...[/bold blue]")
            git_utils.push_branch(branch_name)
            console.print("[green]✓[/green] Pushed branch to remote")

        pr_url = github_client.create_pull_request(
            title=pr_title,
            body=pr_body,
            head=branch_name,
            base=base,
            draft=draft
        )
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
    console.print("[dim]Press Enter to keep the current value[/dim]")

    # JIRA configuration overrides
    console.print("\n[bold]JIRA Configuration[/bold]")
    
    server_url = prompt_with_default("JIRA Server URL", config.jira_server_url)
    config.set_repo('jira.server_url', server_url)

    email = prompt_with_default("JIRA Email", config.jira_email)
    config.set_repo('jira.email', email)

    # Git configuration overrides
    console.print("\n[bold]Git Configuration[/bold]")
    
    branch_pattern = prompt_with_default("Branch Pattern", config.branch_pattern)
    config.set_repo('git.branch_pattern', branch_pattern)

    max_branch_length = prompt_with_default(
        "Max Branch Length",
        config.max_branch_length,
        type=int,
    )
    config.set_repo('git.max_branch_length', max_branch_length)

    # Pull Request configuration overrides
    console.print("\n[bold]Pull Request Configuration[/bold]")
    
    pr_title_pattern = prompt_with_default("PR Title Pattern", config.pr_title_pattern)
    config.set_repo('pull_request.title_pattern', pr_title_pattern)

    pr_draft = prompt_with_default(
        "Create Draft PRs",
        config.pr_draft,
        type=bool,
    )
    config.set_repo('pull_request.draft', pr_draft)

    pr_base_branch = prompt_with_default("PR Base Branch", config.pr_base_branch)
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
    console.print(f"Template: [dim]Stored in config[/dim]")

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
