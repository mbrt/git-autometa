"""
Git operations for git-autometa
"""

import subprocess
import logging
from pathlib import Path
from typing import Optional
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
