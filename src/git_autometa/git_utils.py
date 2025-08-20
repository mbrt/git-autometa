"""
Git operations for git-autometa
"""

import subprocess
import logging
import re
from pathlib import Path
from typing import Optional
import click
from git import Repo
from git.exc import GitCommandError, InvalidGitRepositoryError

logger = logging.getLogger(__name__)


class GitUtils:
    """Git operations utility"""

    def __init__(self, repo_path: Optional[str] = None):
        """
        Initialize GitUtils

        Args:
            repo_path: Path to git repository (optional, uses current directory)
        """
        self.repo_path = Path(repo_path) if repo_path else Path.cwd()
        self.repo = self._get_repository()

    def _get_repository(self) -> Repo:
        """Get git repository"""
        try:
            repo = Repo(self.repo_path, search_parent_directories=True)
            logger.info(f"Found git repository at: {repo.working_dir}")
            return repo
        except InvalidGitRepositoryError:
            raise ValueError(f"No git repository found at {self.repo_path}")

    def get_current_branch(self) -> str:
        """Get current branch name"""
        try:
            return self.repo.active_branch.name
        except Exception as e:
            logger.error(f"Failed to get current branch: {e}")
            raise ValueError(f"Could not determine current branch: {e}")

    def branch_exists(self, branch_name: str) -> bool:
        """
        Check if branch exists locally

        Args:
            branch_name: Branch name to check

        Returns:
            True if branch exists, False otherwise
        """
        try:
            self.repo.heads[branch_name]
            return True
        except IndexError:
            return False

    def create_branch(self, branch_name: str, checkout: bool = True) -> str:
        """
        Create a new git branch

        Args:
            branch_name: Name of the branch to create
            checkout: Whether to checkout the new branch

        Returns:
            Created branch name

        Raises:
            ValueError: If branch already exists or creation fails
        """
        try:
            # Check if branch already exists
            if self.branch_exists(branch_name):
                raise ValueError(f"Branch '{branch_name}' already exists")

            # Ensure we're on a clean working directory
            if self.repo.is_dirty():
                logger.warning("Working directory has uncommitted changes")

            # Create new branch
            new_branch = self.repo.create_head(branch_name)

            if checkout:
                new_branch.checkout()
                logger.info(f"Created and checked out branch: {branch_name}")
            else:
                logger.info(f"Created branch: {branch_name}")

            return branch_name

        except GitCommandError as e:
            logger.error(f"Git command failed: {e}")
            raise ValueError(f"Failed to create branch: {e}")
        except Exception as e:
            logger.error(f"Unexpected error creating branch: {e}")
            raise

    def checkout_branch(self, branch_name: str):
        """
        Checkout an existing branch

        Args:
            branch_name: Branch name to checkout
        """
        try:
            if not self.branch_exists(branch_name):
                raise ValueError(f"Branch '{branch_name}' does not exist")

            self.repo.heads[branch_name].checkout()
            logger.info(f"Checked out branch: {branch_name}")

        except Exception as e:
            logger.error(f"Failed to checkout branch {branch_name}: {e}")
            raise

    def get_remote_url(self, remote_name: str = "origin") -> str:
        """
        Get remote URL

        Args:
            remote_name: Name of the remote

        Returns:
            Remote URL
        """
        try:
            remote = self.repo.remotes[remote_name]
            return remote.url
        except IndexError:
            raise ValueError(f"Remote '{remote_name}' not found")

    def is_branch_pushed(self, branch_name: str) -> bool:
        """
        Check if a branch has been pushed to the remote.

        Args:
            branch_name: The name of the branch to check.

        Returns:
            True if the branch has a remote tracking branch, False otherwise.
        """
        try:
            branch = self.repo.heads[branch_name]
            return branch.tracking_branch() is not None
        except IndexError:
            # Branch doesn't exist locally, so it can't be pushed
            return False
        except Exception as e:
            logger.error(f"Error checking if branch {branch_name} is pushed: {e}")
            return False

    def push_branch(self, branch_name: str, remote_name: str = "origin", set_upstream: bool = True):
        """
        Push branch to remote

        Args:
            branch_name: Branch to push
            remote_name: Remote name
            set_upstream: Whether to set upstream tracking
        """
        try:
            remote = self.repo.remotes[remote_name]

            if set_upstream:
                # Push with upstream tracking
                remote.push(f"{branch_name}:{branch_name}", set_upstream=True)
                logger.info(
                    f"Pushed branch {branch_name} to {remote_name} with upstream tracking")
            else:
                remote.push(branch_name)
                logger.info(f"Pushed branch {branch_name} to {remote_name}")

        except Exception as e:
            logger.error(f"Failed to push branch {branch_name}: {e}")
            raise ValueError(f"Failed to push branch: {e}")

    def is_clean_working_directory(self) -> bool:
        """Check if working directory is clean"""
        return not self.repo.is_dirty()

    def get_status(self) -> str:
        """Get git status as string"""
        try:
            result = subprocess.run(['git', 'status', '--porcelain'],
                                    capture_output=True, text=True, cwd=self.repo.working_dir)
            return result.stdout.strip()
        except Exception as e:
            logger.error(f"Failed to get git status: {e}")
            return ""

    def get_uncommitted_changes(self) -> list[str]:
        """Get list of uncommitted changes"""
        status = self.get_status()
        if not status:
            return []

        changes = []
        for line in status.split('\n'):
            if line.strip():
                # Parse git status format: XY filename
                status_code = line[:2]
                filename = line[3:]
                changes.append(f"{status_code} {filename}")

        return changes

    def stash_changes(self, message: str = "git-autometa: temporary stash") -> bool:
        """
        Stash uncommitted changes

        Args:
            message: Stash message

        Returns:
            True if changes were stashed, False if no changes to stash
        """
        try:
            if not self.repo.is_dirty():
                return False

            self.repo.git.stash('push', '-m', message)
            logger.info(f"Stashed changes: {message}")
            return True

        except Exception as e:
            logger.error(f"Failed to stash changes: {e}")
            raise ValueError(f"Failed to stash changes: {e}")

    def pop_stash(self):
        """Pop the most recent stash"""
        try:
            self.repo.git.stash('pop')
            logger.info("Popped stash")
        except Exception as e:
            logger.error(f"Failed to pop stash: {e}")
            raise ValueError(f"Failed to pop stash: {e}")

    def fetch_remote(self, remote_name: str = "origin"):
        """
        Fetch latest information from remote

        Args:
            remote_name: Name of the remote to fetch from
        """
        try:
            remote = self.repo.remotes[remote_name]
            remote.fetch()
            logger.info(f"Fetched latest information from {remote_name}")
        except Exception as e:
            logger.error(f"Failed to fetch from remote {remote_name}: {e}")
            raise ValueError(f"Failed to fetch from remote: {e}")

    def pull_branch(self, branch_name: str, remote_name: str = "origin"):
        """
        Pull latest changes for a specific branch

        Args:
            branch_name: Name of the branch to pull
            remote_name: Name of the remote to pull from
        """
        try:
            # Ensure we're on the target branch
            current_branch = self.get_current_branch()
            if current_branch != branch_name:
                if self.branch_exists(branch_name):
                    self.checkout_branch(branch_name)
                elif self.remote_branch_exists(branch_name, remote_name):
                    self.checkout_remote_branch(branch_name, remote_name)
                    return  # checkout_remote_branch already gets latest
                else:
                    raise ValueError(f"Branch '{branch_name}' does not exist locally or remotely")

            # Pull latest changes
            remote = self.repo.remotes[remote_name]
            remote.pull(branch_name)
            logger.info(f"Pulled latest changes for branch {branch_name} from {remote_name}")

        except Exception as e:
            logger.error(f"Failed to pull branch {branch_name}: {e}")
            raise ValueError(f"Failed to pull branch: {e}")

    def remote_branch_exists(self, branch_name: str, remote_name: str = "origin") -> bool:
        """
        Check if branch exists on remote

        Args:
            branch_name: Branch name to check
            remote_name: Name of the remote

        Returns:
            True if branch exists on remote, False otherwise
        """
        try:
            remote = self.repo.remotes[remote_name]
            remote_branch_name = f"{remote_name}/{branch_name}"

            # Check if remote branch exists in references
            for ref in remote.refs:
                if ref.name == remote_branch_name:
                    return True
            return False
        except Exception as e:
            logger.error(f"Error checking remote branch {branch_name}: {e}")
            return False

    def delete_local_branch(self, branch_name: str, force: bool = False):
        """
        Delete local branch safely

        Args:
            branch_name: Branch name to delete
            force: Whether to force delete (use with caution)
        """
        try:
            if not self.branch_exists(branch_name):
                logger.warning(f"Branch '{branch_name}' does not exist locally")
                return

            # Don't delete if currently on this branch
            if self.get_current_branch() == branch_name:
                raise ValueError(f"Cannot delete branch '{branch_name}' - currently checked out")

            # Delete the branch
            self.repo.delete_head(branch_name, force=force)
            logger.info(f"Deleted local branch: {branch_name}")

        except Exception as e:
            logger.error(f"Failed to delete local branch {branch_name}: {e}")
            raise ValueError(f"Failed to delete local branch: {e}")

    def get_main_branch_name(self) -> str:
        """
        Determine the main branch name (main or master)

        Returns:
            Name of the main branch
        """
        try:
            # First try 'main'
            if self.branch_exists('main'):
                return 'main'

            # Fallback to 'master'
            if self.branch_exists('master'):
                return 'master'

            # Check remote branches
            try:
                if self.remote_branch_exists('main'):
                    return 'main'
                if self.remote_branch_exists('master'):
                    return 'master'
            except:
                pass

            # Default fallback
            return 'main'

        except Exception as e:
            logger.error(f"Error determining main branch: {e}")
            return 'main'

    def checkout_remote_branch(self, branch_name: str, remote_name: str = "origin"):
        """
        Checkout a branch that exists on remote

        Args:
            branch_name: Branch name to checkout
            remote_name: Name of the remote
        """
        try:
            if not self.remote_branch_exists(branch_name, remote_name):
                raise ValueError(f"Branch '{branch_name}' does not exist on remote '{remote_name}'")

            # Create local branch tracking the remote
            remote_ref = f"{remote_name}/{branch_name}"
            new_branch = self.repo.create_head(branch_name, remote_ref)
            new_branch.set_tracking_branch(self.repo.remotes[remote_name].refs[branch_name])
            new_branch.checkout()

            logger.info(f"Checked out remote branch: {branch_name}")

        except Exception as e:
            logger.error(f"Failed to checkout remote branch {branch_name}: {e}")
            raise ValueError(f"Failed to checkout remote branch: {e}")

    def generate_alternative_branch_name(self, base_name: str) -> str:
        """
        Generate alternative branch name with auto-increment

        Args:
            base_name: Base branch name

        Returns:
            Alternative branch name (e.g., base_name-2, base_name-3)
        """
        counter = 2
        while True:
            alternative_name = f"{base_name}-{counter}"
            if not self.branch_exists(alternative_name) and not self.remote_branch_exists(alternative_name):
                return alternative_name
            counter += 1

            # Safety check to prevent infinite loop
            if counter > 100:
                raise ValueError("Too many branch name conflicts - unable to generate alternative")

    def find_available_branch_name(self, base_name: str) -> str:
        """
        Find first available branch name (base or alternative)

        Args:
            base_name: Base branch name to try

        Returns:
            Available branch name
        """
        if not self.branch_exists(base_name) and not self.remote_branch_exists(base_name):
            return base_name

        return self.generate_alternative_branch_name(base_name)

    def _ensure_latest_main_branch(self):
        """
        Ensure we're on the latest version of the main branch

        This method:
        1. Determines the main branch name (main or master)
        2. Switches to the main branch
        3. Pulls the latest changes from remote
        """
        try:
            main_branch = self.get_main_branch_name()
            logger.info(f"Using main branch: {main_branch}")

            # Pull latest changes for the main branch
            self.pull_branch(main_branch)
            logger.info(f"Updated to latest {main_branch} branch")

        except Exception as e:
            logger.error(f"Failed to ensure latest main branch: {e}")
            raise ValueError(f"Failed to update main branch: {e}")

    def prepare_work_branch(self, base_branch_name: str) -> str:
        """
        Prepare branch for work, starting from latest main branch

        Args:
            base_branch_name: Desired branch name

        Returns:
            Final branch name that was created/checked out
        """
        # Fetch latest remote information
        try:
            self.fetch_remote()
        except:
            logger.warning("Failed to fetch remote - continuing with local information")

        # Ensure we start from the latest main branch
        self._ensure_latest_main_branch()

        local_exists = self.branch_exists(base_branch_name)
        remote_exists = self.remote_branch_exists(base_branch_name)

        # Case 1: Neither exists - create new from current main
        if not local_exists and not remote_exists:
            self.create_branch(base_branch_name, checkout=True)
            return base_branch_name

        # Cases 2, 3, 4: Some form of conflict exists - prompt user
        return self._handle_branch_conflict(base_branch_name, local_exists, remote_exists)

    def _handle_branch_conflict(self, branch_name: str, local_exists: bool, remote_exists: bool) -> str:
        """
        Handle branch conflicts with user prompts

        Args:
            branch_name: Original branch name
            local_exists: Whether branch exists locally
            remote_exists: Whether branch exists on remote

        Returns:
            Final branch name used
        """
        # Determine conflict type for messaging
        if local_exists and remote_exists:
            location = "locally and remotely"
        elif local_exists:
            location = "locally"
        else:
            location = "remotely"

        click.echo(click.style(f"\nBranch '{branch_name}' already exists {location}", fg="yellow"))

        # Show options
        click.echo("\nChoose an action:")
        click.echo("1. Switch to existing branch")
        click.echo("2. Create new branch with alternative name")

        choice = click.prompt("Enter your choice", type=click.Choice(['1', '2']))

        if choice == '1':
            # Switch to existing branch
            if local_exists:
                self.checkout_branch(branch_name)
            else:
                # Remote only - checkout remote branch
                self.checkout_remote_branch(branch_name)
            return branch_name
        else:
            # Create alternative branch
            alternative_name = self.generate_alternative_branch_name(branch_name)
            click.echo(f"Creating alternative branch: {alternative_name}")
            self.create_branch(alternative_name, checkout=True)
            return alternative_name

    def get_commit_messages_for_pr(self, base_branch: str) -> str:
        """
        Get formatted commit messages for commits that will be included in the PR

        Args:
            base_branch: The base branch to compare against

        Returns:
            Formatted string of commit messages as bulleted list
        """
        try:
            current_branch = self.get_current_branch()

            # Get commits that are in current branch but not in base branch
            commit_range = f"{base_branch}..{current_branch}"

            # Get commit messages using git log
            result = subprocess.run([
                'git', 'log', 
                '--pretty=format:%s',  # Just the subject line
                '--reverse',           # Show oldest first
                commit_range
            ], capture_output=True, text=True, cwd=self.repo.working_dir)

            if result.returncode != 0:
                logger.warning(f"Failed to get commit messages: {result.stderr}")
                return ""

            commit_messages = result.stdout.strip()
            if not commit_messages:
                logger.info("No commits found for PR")
                return ""

            # Format as bulleted list and remove JIRA tags
            formatted_messages = []
            # Pattern to match JIRA tags like [ABC-123] at the beginning of commit messages
            jira_tag_pattern = r'^\[([A-Z]+-\d+)\]\s*'

            for message in commit_messages.split('\n'):
                if message.strip():  # Skip empty lines
                    # Remove JIRA tag if present
                    cleaned_message = re.sub(jira_tag_pattern, '', message.strip())
                    formatted_messages.append(f"* {cleaned_message}")

            if formatted_messages:
                return '\n'.join(formatted_messages)
            else:
                return ""

        except Exception as e:
            logger.error(f"Error getting commit messages for PR: {e}")
            return ""
