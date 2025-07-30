"""
GitHub CLI wrapper for git-autometa
"""

import subprocess
import json
import logging
from typing import Optional, Dict, Any, List

logger = logging.getLogger(__name__)


class GitHubClient:
    """GitHub CLI wrapper"""
    
    def __init__(self):
        """Initialize GitHub CLI wrapper"""
        self._check_gh_cli()
        
    def _check_gh_cli(self):
        """Check if GitHub CLI is installed and authenticated"""
        try:
            # Check if gh is installed
            subprocess.run(['gh', '--version'], 
                         capture_output=True, check=True, text=True)
            
            # Check if authenticated
            result = subprocess.run(['gh', 'auth', 'status'], 
                                  capture_output=True, text=True)
            if result.returncode != 0:
                raise ValueError("GitHub CLI not authenticated. Please run 'gh auth login'")
                
            logger.info("GitHub CLI is installed and authenticated")
            
        except FileNotFoundError:
            raise ValueError("GitHub CLI not found. Please install 'gh' CLI tool")
        except subprocess.CalledProcessError:
            raise ValueError("GitHub CLI not available")
            
    def test_connection(self) -> bool:
        """Test GitHub connection"""
        try:
            result = subprocess.run(['gh', 'api', 'user'], 
                                  capture_output=True, check=True, text=True)
            user_data = json.loads(result.stdout)
            logger.info(f"GitHub connection test successful for user: {user_data.get('login')}")
            return True
        except Exception as e:
            logger.error(f"GitHub connection test failed: {e}")
            return False
            
    def get_current_repo(self) -> tuple[str, str]:
        """
        Get current repository owner and name
        
        Returns:
            Tuple of (owner, repo)
        """
        try:
            result = subprocess.run(['gh', 'repo', 'view', '--json', 'owner,name'], 
                                  capture_output=True, check=True, text=True)
            repo_data = json.loads(result.stdout)
            owner = repo_data['owner']['login']
            repo = repo_data['name']
            logger.info(f"Current repository: {owner}/{repo}")
            return owner, repo
        except Exception as e:
            logger.error(f"Failed to get current repository: {e}")
            raise ValueError(f"Could not determine current repository: {e}")
            
    def get_default_branch(self) -> str:
        """
        Get repository's default branch
        
        Returns:
            Default branch name
        """
        try:
            result = subprocess.run(['gh', 'repo', 'view', '--json', 'defaultBranchRef'], 
                                  capture_output=True, check=True, text=True)
            repo_data = json.loads(result.stdout)
            default_branch = repo_data['defaultBranchRef']['name']
            logger.info(f"Default branch: {default_branch}")
            return default_branch
        except Exception as e:
            logger.error(f"Failed to get default branch: {e}")
            return "main"  # Fallback to main
            
    def branch_exists(self, branch_name: str) -> bool:
        """
        Check if branch exists in repository
        
        Args:
            branch_name: Branch name to check
            
        Returns:
            True if branch exists, False otherwise
        """
        try:
            result = subprocess.run(['gh', 'api', f'repos/{{owner}}/{{repo}}/branches/{branch_name}'], 
                                  capture_output=True, text=True)
            return result.returncode == 0
        except:
            return False
            
    def create_pull_request(
        self,
        title: str,
        body: str,
        head: str,
        base: str = "",
        draft: bool = True
    ) -> str:
        """
        Create a pull request
        
        Args:
            title: PR title
            body: PR description
            head: Head branch name
            base: Base branch name (optional, uses default branch)
            draft: Whether to create as draft
            
        Returns:
            PR URL
        """
        try:
            cmd = ['gh', 'pr', 'create', '--title', title, '--body', body]
            
            if base:
                cmd.extend(['--base', base])
                
            if draft:
                cmd.append('--draft')
                
            result = subprocess.run(cmd, capture_output=True, check=True, text=True)
            pr_url = result.stdout.strip()
            
            logger.info(f"Created pull request: {pr_url}")
            return pr_url
            
        except subprocess.CalledProcessError as e:
            error_msg = e.stderr.strip() if e.stderr else str(e)
            logger.error(f"Failed to create pull request: {error_msg}")
            raise ValueError(f"Failed to create pull request: {error_msg}")
            
    def list_pull_requests(self, state: str = "open") -> List[Dict[str, Any]]:
        """
        List pull requests
        
        Args:
            state: PR state (open, closed, merged, all)
            
        Returns:
            List of PR data
        """
        try:
            result = subprocess.run([
                'gh', 'pr', 'list', 
                '--state', state,
                '--json', 'number,title,headRefName,baseRefName,url,draft'
            ], capture_output=True, check=True, text=True)
            
            return json.loads(result.stdout)
            
        except Exception as e:
            logger.error(f"Failed to list pull requests: {e}")
            return []
            
    def get_pull_request_for_branch(self, branch_name: str) -> Optional[Dict[str, Any]]:
        """
        Get pull request for a specific branch
        
        Args:
            branch_name: Branch name
            
        Returns:
            PR data if found, None otherwise
        """
        prs = self.list_pull_requests(state="all")
        for pr in prs:
            if pr['headRefName'] == branch_name:
                return pr
        return None
