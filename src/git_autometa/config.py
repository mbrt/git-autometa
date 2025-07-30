"""
Configuration management for git-autometa
"""

import yaml
import re
from pathlib import Path
from typing import Dict, Any, Optional
import logging
from urllib.parse import urlparse

from .git_utils import GitUtils

logger = logging.getLogger(__name__)


class Config:
    """Configuration manager for git-autometa"""

    def __init__(self, config_path: Optional[str] = None):
        """
        Initialize configuration

        Args:
            config_path: Path to configuration file. If None, uses centralized config system.
        """
        self.config_data = {}
        self.repo_config_data = {}
        self.custom_config_path = config_path
        
        if config_path:
            # Use custom config path if provided
            self.config_path = Path(config_path)
            self._load_custom_config()
        else:
            # Use centralized configuration system
            self.config_dir = self._get_config_dir()
            self.global_config_path = self.config_dir / "config.yaml"
            self._load_centralized_config()

    def _get_config_dir(self) -> Path:
        """Get centralized configuration directory"""
        # Use XDG config directory
        config_home = Path.home() / ".config" / "git-autometa"
        config_home.mkdir(parents=True, exist_ok=True)
        return config_home

    def _get_repo_identifier(self) -> Optional[str]:
        """
        Get repository identifier from git remote origin
        
        Returns:
            Repository identifier in format 'owner_repo' or None if not a git repo
        """
        try:
            git_utils = GitUtils()
            remote_url = git_utils.get_remote_url("origin")
            return self._parse_github_repo_from_url(remote_url)
        except Exception as e:
            logger.debug(f"Could not determine repository identifier: {e}")
            return None

    def _parse_github_repo_from_url(self, url: str) -> Optional[str]:
        """
        Parse GitHub repository owner/repo from various URL formats
        
        Args:
            url: Git remote URL (HTTPS or SSH)
            
        Returns:
            Repository identifier in format 'owner_repo' or None
        """
        # Handle SSH URLs: git@github.com:owner/repo.git
        ssh_match = re.match(r'git@github\.com:([^/]+)/(.+?)(?:\.git)?$', url)
        if ssh_match:
            owner, repo = ssh_match.groups()
            return f"{owner}_{repo}"
        
        # Handle HTTPS URLs: https://github.com/owner/repo.git
        try:
            parsed = urlparse(url)
            if parsed.hostname == 'github.com':
                path_parts = parsed.path.strip('/').split('/')
                if len(path_parts) >= 2:
                    owner = path_parts[0]
                    repo = path_parts[1].rstrip('.git')
                    return f"{owner}_{repo}"
        except Exception:
            pass
        
        logger.warning(f"Could not parse GitHub repository from URL: {url}")
        return None

    def _load_custom_config(self):
        """Load configuration from custom path"""
        try:
            if self.config_path.exists():
                with open(self.config_path, 'r') as f:
                    self.config_data = yaml.safe_load(f) or {}
                logger.info(f"Loaded custom configuration from {self.config_path}")
            else:
                logger.warning(f"Custom configuration file not found: {self.config_path}")
                self.config_data = {}
        except Exception as e:
            logger.error(f"Error loading custom configuration: {e}")
            self.config_data = {}

    def _load_centralized_config(self):
        """Load configuration from centralized system"""
        # Load global config
        self._load_global_config()
        
        # Load repository-specific config if in a git repository
        repo_id = self._get_repo_identifier()
        if repo_id:
            self._load_repo_config(repo_id)
            logger.info(f"Using configuration for repository: {repo_id.replace('_', '/')}")

    def _load_global_config(self):
        """Load global configuration"""
        try:
            if self.global_config_path.exists():
                with open(self.global_config_path, 'r') as f:
                    self.config_data = yaml.safe_load(f) or {}
                logger.debug(f"Loaded global configuration from {self.global_config_path}")
            else:
                logger.debug("No global configuration file found, using defaults")
                self.config_data = {}
        except Exception as e:
            logger.error(f"Error loading global configuration: {e}")
            self.config_data = {}

    def _load_repo_config(self, repo_id: str):
        """Load repository-specific configuration"""
        try:
            repo_config_path = self.config_dir / "repositories" / f"{repo_id}.yaml"
            if repo_config_path.exists():
                with open(repo_config_path, 'r') as f:
                    self.repo_config_data = yaml.safe_load(f) or {}
                logger.debug(f"Loaded repository configuration from {repo_config_path}")
            else:
                logger.debug(f"No repository-specific configuration found for {repo_id}")
                self.repo_config_data = {}
        except Exception as e:
            logger.error(f"Error loading repository configuration: {e}")
            self.repo_config_data = {}

    def get(self, key: str, default: Any = None) -> Any:
        """
        Get configuration value using dot notation with hierarchy resolution
        
        Priority order:
        1. Repository-specific config
        2. Global config
        3. Default value

        Args:
            key: Configuration key (e.g., 'jira.server_url')
            default: Default value if key not found

        Returns:
            Configuration value
        """
        # Try repository-specific config first
        if self.repo_config_data:
            value = self._get_nested_value(self.repo_config_data, key)
            if value is not None:
                return value

        # Try global config
        value = self._get_nested_value(self.config_data, key)
        if value is not None:
            return value

        return default

    def _get_nested_value(self, data: Dict[str, Any], key: str) -> Any:
        """Get nested value from dictionary using dot notation"""
        keys = key.split('.')
        value = data

        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return None

        return value

    def set_global(self, key: str, value: Any):
        """
        Set global configuration value using dot notation

        Args:
            key: Configuration key (e.g., 'jira.server_url')
            value: Value to set
        """
        self._set_nested_value(self.config_data, key, value)

    def set_repo(self, key: str, value: Any):
        """
        Set repository-specific configuration value using dot notation

        Args:
            key: Configuration key (e.g., 'jira.server_url')
            value: Value to set
        """
        repo_id = self._get_repo_identifier()
        if not repo_id:
            raise ValueError("Not in a git repository with GitHub remote")
        
        self._set_nested_value(self.repo_config_data, key, value)

    def _set_nested_value(self, data: Dict[str, Any], key: str, value: Any):
        """Set nested value in dictionary using dot notation"""
        keys = key.split('.')
        config = data

        for k in keys[:-1]:
            if k not in config:
                config[k] = {}
            config = config[k]

        config[keys[-1]] = value

    def save_global(self):
        """Save global configuration to file"""
        try:
            self.global_config_path.parent.mkdir(parents=True, exist_ok=True)
            with open(self.global_config_path, 'w') as f:
                yaml.dump(self.config_data, f, default_flow_style=False)
            logger.info(f"Global configuration saved to {self.global_config_path}")
        except Exception as e:
            logger.error(f"Error saving global configuration: {e}")

    def save_repo(self):
        """Save repository-specific configuration to file"""
        repo_id = self._get_repo_identifier()
        if not repo_id:
            raise ValueError("Not in a git repository with GitHub remote")

        try:
            repo_config_dir = self.config_dir / "repositories"
            repo_config_dir.mkdir(parents=True, exist_ok=True)

            repo_config_path = repo_config_dir / f"{repo_id}.yaml"
            with open(repo_config_path, 'w') as f:
                yaml.dump(self.repo_config_data, f, default_flow_style=False)
            logger.info(f"Repository configuration saved to {repo_config_path}")
        except Exception as e:
            logger.error(f"Error saving repository configuration: {e}")

    def save_custom(self):
        """Save configuration to custom path (if using custom config)"""
        if not self.custom_config_path:
            raise ValueError("No custom config path specified")
        
        try:
            self.config_path.parent.mkdir(parents=True, exist_ok=True)
            with open(self.config_path, 'w') as f:
                yaml.dump(self.config_data, f, default_flow_style=False)
            logger.info(f"Configuration saved to {self.config_path}")
        except Exception as e:
            logger.error(f"Error saving configuration: {e}")

    def get_current_repo_id(self) -> Optional[str]:
        """Get current repository identifier"""
        return self._get_repo_identifier()

    @property
    def config_path_info(self) -> str:
        """Get information about configuration paths being used"""
        if self.custom_config_path:
            return f"Custom: {self.config_path}"
        
        info = f"Global: {self.global_config_path}"
        repo_id = self._get_repo_identifier()
        if repo_id:
            repo_config_path = self.config_dir / "repositories" / f"{repo_id}.yaml"
            info += f"\nRepository: {repo_config_path}"
        
        return info

    # Convenience properties for common config values
    @property
    def jira_server_url(self) -> str:
        """Get JIRA server URL"""
        return self.get('jira.server_url', '')

    @property
    def jira_email(self) -> str:
        """Get JIRA email"""
        return self.get('jira.email', '')

    @property
    def github_owner(self) -> str:
        """Get GitHub owner"""
        return self.get('github.owner', '')

    @property
    def github_repo(self) -> str:
        """Get GitHub repository"""
        return self.get('github.repo', '')

    @property
    def branch_pattern(self) -> str:
        """Get branch naming pattern"""
        return self.get('git.branch_pattern', 'feature/{jira_id}-{jira_title}')

    @property
    def max_branch_length(self) -> int:
        """Get maximum branch length"""
        return self.get('git.max_branch_length', 50)

    @property
    def pr_title_pattern(self) -> str:
        """Get PR title pattern"""
        return self.get('pull_request.title_pattern', '{jira_id}: {jira_title}')

    @property
    def pr_draft(self) -> bool:
        """Get whether to create draft PRs"""
        return self.get('pull_request.draft', True)

    @property
    def pr_base_branch(self) -> str:
        """Get PR base branch"""
        return self.get('pull_request.base_branch', 'main')

    @property
    def pr_template(self) -> str:
        """Get PR template content"""
        default_template = """
    # What this Pull Request does/why we need it

    {jira_description}

    ## What type of PR is this?

    feature

    ## Relevant links

    * [{jira_id}]({jira_url})"""
        return self.get('pull_request.template', default_template)

    @property
    def log_level(self) -> str:
        """Get log level"""
        return self.get('log_level', 'INFO')
