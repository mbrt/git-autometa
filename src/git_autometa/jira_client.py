"""
JIRA API client for git-autometa
"""

import requests
import keyring
import logging
import re
from typing import Dict, Any, Optional, List
from urllib.parse import urljoin

from .jira_markdown_converter import convert_jira_to_markdown

logger = logging.getLogger(__name__)


class JiraIssue:
    """Represents a JIRA issue"""

    def __init__(self, data: Dict[str, Any]):
        self.data = data

    @property
    def key(self) -> str:
        """Issue key (e.g., PROJ-123)"""
        return self.data.get('key', '')

    @property
    def summary(self) -> str:
        """Issue summary/title"""
        return self.data.get('fields', {}).get('summary', '')

    @property
    def description(self) -> str:
        """Issue description"""
        return self.data.get('fields', {}).get('description', '')

    @property
    def description_markdown(self) -> str:
        """Issue description converted to Markdown"""
        raw_description = self.description
        return convert_jira_to_markdown(raw_description)

    @property
    def issue_type(self) -> str:
        """Issue type (e.g., Bug, Feature, Task)"""
        issue_type = self.data.get('fields', {}).get('issuetype', {})
        return issue_type.get('name', '').lower()

    @property
    def status(self) -> str:
        """Issue status"""
        status = self.data.get('fields', {}).get('status', {})
        return status.get('name', '')

    @property
    def assignee(self) -> Optional[str]:
        """Issue assignee"""
        assignee = self.data.get('fields', {}).get('assignee')
        if assignee:
            return assignee.get('displayName') or assignee.get('name')
        return None

    @property
    def url(self) -> str:
        """Issue URL (will be set by JiraClient)"""
        return getattr(self, '_url', '')

    def set_url(self, base_url: str):
        """Set the issue URL based on base URL"""
        self._url = urljoin(base_url, f"/browse/{self.key}")

    def slugify_title(self, max_length: int = 30) -> str:
        """Convert title to slug for branch names"""
        # Remove non-alphanumeric characters and convert to lowercase
        slug = re.sub(r'[^a-zA-Z0-9\s-]', '', self.summary.lower())
        # Replace spaces with hyphens
        slug = re.sub(r'\s+', '-', slug)
        # Remove multiple consecutive hyphens
        slug = re.sub(r'-+', '-', slug)
        # Trim hyphens from ends
        slug = slug.strip('-')
        # Truncate to max length
        if len(slug) > max_length:
            slug = slug[:max_length].rstrip('-')
        return slug


class JiraClient:
    """JIRA API client"""

    def __init__(self, server_url: str, email: str):
        """
        Initialize JIRA client

        Args:
            server_url: JIRA server URL
            email: User email for authentication
        """
        self.server_url = server_url.rstrip('/')
        self.email = email
        self.session = requests.Session()

        # Get API token from keyring
        self._setup_authentication()

    def _setup_authentication(self):
        """Setup authentication using keyring"""
        # Try to get token from keyring first
        api_token = keyring.get_password("git-autometa-jira", self.email)

        if not api_token:
            logger.warning(
                f"No JIRA API token found in keyring for {self.email}")
            logger.info(
                "Please run 'git-autometa config' to set up your JIRA credentials")
            raise ValueError(
                "JIRA API token not found. Please configure your credentials.")

        # Set up basic auth with email and API token
        self.session.auth = (self.email, api_token)
        self.session.headers.update({
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        })

    @classmethod
    def store_api_token(cls, email: str, api_token: str):
        """Store API token in keyring"""
        try:
            keyring.set_password("git-autometa-jira", email, api_token)
            logger.info(f"JIRA API token stored securely for {email}")
        except Exception as e:
            logger.error(f"Failed to store JIRA API token: {e}")
            raise

    def test_connection(self) -> bool:
        """Test JIRA connection"""
        try:
            url = urljoin(self.server_url, '/rest/api/2/myself')
            response = self.session.get(url, timeout=10)
            response.raise_for_status()
            logger.info("JIRA connection test successful")
            return True
        except Exception as e:
            logger.error(f"JIRA connection test failed: {e}")
            return False

    def search_my_issues(self, limit: int = 15) -> List[JiraIssue]:
        """
        Search for issues assigned to the current user

        Args:
            limit: Maximum number of issues to return

        Returns:
            List of JiraIssue objects ordered by update date (most recent first)

        Raises:
            ValueError: If search fails
        """
        try:
            # JQL to find issues assigned to current user, ordered by most recent first
            jql = "assignee = currentUser() ORDER BY updated DESC"

            url = urljoin(self.server_url, '/rest/api/2/search')
            params = {
                'jql': jql,
                'maxResults': limit,
                'fields': 'summary,description,issuetype,status,assignee'
            }

            response = self.session.get(url, params=params, timeout=10)
            response.raise_for_status()
            data = response.json()

            issues = []
            for issue_data in data.get('issues', []):
                issue = JiraIssue(issue_data)
                issue.set_url(self.server_url)
                issues.append(issue)

            logger.info(f"Found {len(issues)} issues assigned to current user")
            return issues

        except requests.exceptions.RequestException as e:
            logger.error(f"Error searching JIRA issues: {e}")
            raise ValueError(f"Failed to search JIRA issues: {e}")
        except Exception as e:
            logger.error(f"Unexpected error searching JIRA issues: {e}")
            raise

    def get_issue(self, issue_key: str) -> JiraIssue:
        """
        Get JIRA issue by key

        Args:
            issue_key: JIRA issue key (e.g., PROJ-123)

        Returns:
            JiraIssue object

        Raises:
            ValueError: If issue not found or API error
        """
        try:
            # Clean up issue key
            issue_key = issue_key.strip().upper()

            # Validate issue key format
            if not re.match(r'^[A-Z]+-\d+$', issue_key):
                raise ValueError(f"Invalid JIRA issue key format: {issue_key}")

            url = urljoin(self.server_url, f'/rest/api/2/issue/{issue_key}')
            response = self.session.get(url, timeout=10)

            if response.status_code == 404:
                raise ValueError(f"JIRA issue not found: {issue_key}")

            response.raise_for_status()
            data = response.json()

            issue = JiraIssue(data)
            issue.set_url(self.server_url)

            logger.info(f"Retrieved JIRA issue: {issue_key} - {issue.summary}")
            return issue

        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching JIRA issue {issue_key}: {e}")
            raise ValueError(f"Failed to fetch JIRA issue: {e}")
        except Exception as e:
            logger.error(
                f"Unexpected error fetching JIRA issue {issue_key}: {e}")
            raise
