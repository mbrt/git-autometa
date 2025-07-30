"""
Configuration management for git-autometa
"""

import os
import yaml
from pathlib import Path
from typing import Dict, Any, Optional
import logging

logger = logging.getLogger(__name__)


class Config:
    """Configuration manager for git-autometa"""
    
    def __init__(self, config_path: Optional[str] = None):
        """
        Initialize configuration
        
        Args:
            config_path: Path to configuration file. If None, uses default locations.
        """
        self.config_data = {}
        self.config_path = self._find_config_path(config_path)
        self._load_config()
        
    def _find_config_path(self, config_path: Optional[str]) -> Path:
        """Find configuration file path"""
        if config_path:
            return Path(config_path)
            
        # Check for config in current directory first
        current_dir = Path.cwd()
        local_config = current_dir / "config.yaml"
        if local_config.exists():
            return local_config
            
        # Check for config in git root
        git_root = self._find_git_root()
        if git_root:
            git_config = git_root / "config.yaml"
            if git_config.exists():
                return git_config
                
        # Fall back to package default
        package_dir = Path(__file__).parent.parent.parent
        default_config = package_dir / "config.yaml"
        return default_config
        
    def _find_git_root(self) -> Optional[Path]:
        """Find git repository root"""
        current_path = Path.cwd()
        while current_path != current_path.parent:
            if (current_path / ".git").exists():
                return current_path
            current_path = current_path.parent
        return None
        
    def _load_config(self):
        """Load configuration from file"""
        try:
            if self.config_path.exists():
                with open(self.config_path, 'r') as f:
                    self.config_data = yaml.safe_load(f) or {}
                logger.info(f"Loaded configuration from {self.config_path}")
            else:
                logger.warning(f"Configuration file not found: {self.config_path}")
                self.config_data = {}
        except Exception as e:
            logger.error(f"Error loading configuration: {e}")
            self.config_data = {}
            
    def get(self, key: str, default: Any = None) -> Any:
        """
        Get configuration value using dot notation
        
        Args:
            key: Configuration key (e.g., 'jira.server_url')
            default: Default value if key not found
            
        Returns:
            Configuration value
        """
        keys = key.split('.')
        value = self.config_data
        
        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default
                
        return value
        
    def set(self, key: str, value: Any):
        """
        Set configuration value using dot notation
        
        Args:
            key: Configuration key (e.g., 'jira.server_url')
            value: Value to set
        """
        keys = key.split('.')
        config = self.config_data
        
        for k in keys[:-1]:
            if k not in config:
                config[k] = {}
            config = config[k]
            
        config[keys[-1]] = value
        
    def save(self):
        """Save configuration to file"""
        try:
            self.config_path.parent.mkdir(parents=True, exist_ok=True)
            with open(self.config_path, 'w') as f:
                yaml.dump(self.config_data, f, default_flow_style=False)
            logger.info(f"Configuration saved to {self.config_path}")
        except Exception as e:
            logger.error(f"Error saving configuration: {e}")
            
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
    def pr_template_path(self) -> str:
        """Get PR template path"""
        return self.get('pull_request.template_path', 'templates/pr_template.md')
        
    @property
    def log_level(self) -> str:
        """Get log level"""
        return self.get('log_level', 'INFO')
