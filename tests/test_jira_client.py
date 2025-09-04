"""
Tests for JIRA client functionality
"""

import pytest
import requests
from unittest.mock import Mock, patch
from src.git_autometa.jira_client import JiraClient, JiraIssue


class TestJiraIssue:
    """Test JiraIssue class"""

    def test_issue_properties(self):
        """Test basic issue property extraction"""
        issue_data = {
            'key': 'PROJ-123',
            'fields': {
                'summary': 'Test Issue Summary',
                'description': 'Test description',
                'issuetype': {'name': 'Bug'},
                'status': {'name': 'In Progress'},
                'assignee': {'displayName': 'John Doe', 'name': 'john.doe'}
            }
        }
        
        issue = JiraIssue(issue_data)
        
        assert issue.key == 'PROJ-123'
        assert issue.summary == 'Test Issue Summary'
        assert issue.description == 'Test description'
        assert issue.issue_type == 'bug'
        assert issue.status == 'In Progress'
        assert issue.assignee == 'John Doe'

    def test_issue_missing_fields(self):
        """Test issue with missing optional fields"""
        issue_data = {
            'key': 'PROJ-456',
            'fields': {
                'summary': 'Minimal Issue',
                'issuetype': {'name': 'Task'},
                'status': {'name': 'Open'},
                'assignee': None
            }
        }
        
        issue = JiraIssue(issue_data)
        
        assert issue.key == 'PROJ-456'
        assert issue.summary == 'Minimal Issue'
        assert issue.description == ''
        assert issue.issue_type == 'task'
        assert issue.status == 'Open'
        assert issue.assignee is None

    def test_issue_adf_description(self):
        """Test issue with ADF (Atlassian Document Format) description"""
        # Example ADF structure from JIRA v3 API
        adf_description = {
            "version": 1,
            "type": "doc",
            "content": [
                {
                    "type": "paragraph",
                    "content": [
                        {
                            "type": "text",
                            "text": "This is a paragraph with "
                        },
                        {
                            "type": "text",
                            "text": "bold text",
                            "marks": [{"type": "strong"}]
                        },
                        {
                            "type": "text",
                            "text": " and normal text."
                        }
                    ]
                },
                {
                    "type": "paragraph",
                    "content": [
                        {
                            "type": "text",
                            "text": "Second paragraph."
                        }
                    ]
                }
            ]
        }
        
        issue_data = {
            'key': 'PROJ-789',
            'fields': {
                'summary': 'ADF Issue',
                'description': adf_description,
                'issuetype': {'name': 'Story'},
                'status': {'name': 'Open'},
                'assignee': {'displayName': 'Test User'}
            }
        }
        
        issue = JiraIssue(issue_data)
        
        assert issue.key == 'PROJ-789'
        assert issue.summary == 'ADF Issue'
        # Should extract plain text from ADF
        expected_text = "This is a paragraph with bold text and normal text.\nSecond paragraph."
        assert issue.description == expected_text
        assert issue.issue_type == 'story'
        assert issue.status == 'Open'
        assert issue.assignee == 'Test User'

    def test_issue_plain_text_description(self):
        """Test issue with plain text description (backward compatibility)"""
        issue_data = {
            'key': 'PROJ-101',
            'fields': {
                'summary': 'Plain Text Issue',
                'description': 'This is a plain text description.',
                'issuetype': {'name': 'Bug'},
                'status': {'name': 'In Progress'},
                'assignee': {'displayName': 'Developer'}
            }
        }
        
        issue = JiraIssue(issue_data)
        
        assert issue.key == 'PROJ-101'
        assert issue.summary == 'Plain Text Issue'
        assert issue.description == 'This is a plain text description.'
        assert issue.issue_type == 'bug'
        assert issue.status == 'In Progress'
        assert issue.assignee == 'Developer'

    def test_issue_complex_adf_description(self):
        """Test issue with complex ADF description including lists"""
        adf_description = {
            "version": 1,
            "type": "doc",
            "content": [
                {
                    "type": "heading",
                    "attrs": {"level": 2},
                    "content": [
                        {
                            "type": "text",
                            "text": "Requirements"
                        }
                    ]
                },
                {
                    "type": "bulletList",
                    "content": [
                        {
                            "type": "listItem",
                            "content": [
                                {
                                    "type": "paragraph",
                                    "content": [
                                        {
                                            "type": "text",
                                            "text": "First requirement"
                                        }
                                    ]
                                }
                            ]
                        },
                        {
                            "type": "listItem",
                            "content": [
                                {
                                    "type": "paragraph",
                                    "content": [
                                        {
                                            "type": "text",
                                            "text": "Second requirement"
                                        }
                                    ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
        
        issue_data = {
            'key': 'PROJ-202',
            'fields': {
                'summary': 'Complex ADF Issue',
                'description': adf_description,
                'issuetype': {'name': 'Epic'},
                'status': {'name': 'Planning'},
                'assignee': {'displayName': 'Product Owner'}
            }
        }
        
        issue = JiraIssue(issue_data)
        
        assert issue.key == 'PROJ-202'
        assert issue.summary == 'Complex ADF Issue'
        # Should extract text from headings and lists
        description = issue.description
        assert 'Requirements' in description
        assert '• First requirement' in description
        assert '• Second requirement' in description

    def test_issue_url_setting(self):
        """Test setting issue URL"""
        issue_data = {'key': 'PROJ-789', 'fields': {}}
        issue = JiraIssue(issue_data)
        
        issue.set_url('https://test.atlassian.net')
        assert issue.url == 'https://test.atlassian.net/browse/PROJ-789'

    def test_slugify_title(self):
        """Test title slugification for branch names"""
        test_cases = [
            ('Fix bug in user authentication', 'fix-bug-in-user-authentication'),
            ('Update API documentation!', 'update-api-documentation'),
            ('Feature: Add new payment method', 'feature-add-new-payment-method'),
            ('This is a very long title that should be truncated appropriately', 'this-is-a-very-long-title-that'),
            ('Multiple    spaces   and-dashes', 'multiple-spaces-and-dashes'),
            ('Title with (special) chars & symbols!', 'title-with-special-chars-symbo'),
        ]
        
        for title, expected_slug in test_cases:
            issue_data = {'key': 'TEST-1', 'fields': {'summary': title}}
            issue = JiraIssue(issue_data)
            assert issue.slugify_title() == expected_slug


class TestJiraClient:
    """Test JiraClient class"""

    @patch('keyring.get_password')
    def test_client_initialization_success(self, mock_get_password):
        """Test successful client initialization"""
        mock_get_password.return_value = 'test-api-token'
        
        client = JiraClient('https://test.atlassian.net', 'test@example.com')
        
        assert client.server_url == 'https://test.atlassian.net'
        assert client.email == 'test@example.com'
        assert client.session.auth == ('test@example.com', 'test-api-token')
        mock_get_password.assert_called_once_with("git-autometa-jira", "test@example.com")

    @patch('keyring.get_password')
    def test_client_initialization_no_token(self, mock_get_password):
        """Test client initialization without API token"""
        mock_get_password.return_value = None
        
        with pytest.raises(ValueError, match="JIRA API token not found"):
            JiraClient('https://test.atlassian.net', 'test@example.com')

    @patch('keyring.set_password')
    def test_store_api_token_success(self, mock_set_password):
        """Test API token storage"""
        JiraClient.store_api_token('test@example.com', 'test-token')
        mock_set_password.assert_called_once_with("git-autometa-jira", "test@example.com", "test-token")

    @patch('keyring.set_password')
    def test_store_api_token_failure(self, mock_set_password):
        """Test API token storage failure"""
        mock_set_password.side_effect = Exception("Keyring error")
        
        with pytest.raises(Exception, match="Keyring error"):
            JiraClient.store_api_token('test@example.com', 'test-token')

    @patch('keyring.get_password')
    def test_connection_test_success(self, mock_get_password):
        """Test successful connection test"""
        mock_get_password.return_value = 'test-api-token'
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.raise_for_status.return_value = None
            mock_get.return_value = mock_response
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            result = client.test_connection()
            
            assert result is True
            mock_get.assert_called_once()
            args, kwargs = mock_get.call_args
            assert '/rest/api/3/myself' in args[0]

    @patch('keyring.get_password')
    def test_connection_test_failure(self, mock_get_password):
        """Test connection test failure"""
        mock_get_password.return_value = 'test-api-token'
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_get.side_effect = requests.exceptions.RequestException("Connection failed")
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            result = client.test_connection()
            
            assert result is False

    @patch('keyring.get_password')
    def test_search_my_issues_success(self, mock_get_password):
        """Test successful issue search"""
        mock_get_password.return_value = 'test-api-token'
        
        mock_response_data = {
            'issues': [
                {
                    'key': 'PROJ-123',
                    'fields': {
                        'summary': 'Test Issue 1',
                        'description': 'Description 1',
                        'issuetype': {'name': 'Bug'},
                        'status': {'name': 'Open'},
                        'assignee': {'displayName': 'John Doe'}
                    }
                },
                {
                    'key': 'PROJ-456',
                    'fields': {
                        'summary': 'Test Issue 2',
                        'description': 'Description 2',
                        'issuetype': {'name': 'Task'},
                        'status': {'name': 'In Progress'},
                        'assignee': {'displayName': 'Jane Smith'}
                    }
                }
            ]
        }
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.raise_for_status.return_value = None
            mock_response.json.return_value = mock_response_data
            mock_get.return_value = mock_response
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            issues = client.search_my_issues(limit=10)
            
            assert len(issues) == 2
            assert issues[0].key == 'PROJ-123'
            assert issues[0].summary == 'Test Issue 1'
            assert issues[1].key == 'PROJ-456'
            assert issues[1].summary == 'Test Issue 2'
            
            # Verify the API v3 endpoint is used
            args, kwargs = mock_get.call_args
            assert '/rest/api/3/search/jql' in args[0]
            assert 'jql' in kwargs['params']
            assert kwargs['params']['maxResults'] == 10

    @patch('keyring.get_password')
    def test_search_my_issues_api_error(self, mock_get_password):
        """Test issue search API error"""
        mock_get_password.return_value = 'test-api-token'
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_get.side_effect = requests.exceptions.RequestException("API Error")
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            
            with pytest.raises(ValueError, match="Failed to search JIRA issues"):
                client.search_my_issues()

    @patch('keyring.get_password')
    def test_get_issue_success(self, mock_get_password):
        """Test successful individual issue fetch"""
        mock_get_password.return_value = 'test-api-token'
        
        mock_response_data = {
            'key': 'PROJ-123',
            'fields': {
                'summary': 'Individual Issue',
                'description': 'Issue description',
                'issuetype': {'name': 'Story'},
                'status': {'name': 'Done'},
                'assignee': {'displayName': 'Test User'}
            }
        }
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.raise_for_status.return_value = None
            mock_response.json.return_value = mock_response_data
            mock_get.return_value = mock_response
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            issue = client.get_issue('PROJ-123')
            
            assert issue.key == 'PROJ-123'
            assert issue.summary == 'Individual Issue'
            assert issue.status == 'Done'
            
            # Verify the API v3 endpoint is used
            args, kwargs = mock_get.call_args
            assert '/rest/api/3/issue/PROJ-123' in args[0]

    @patch('keyring.get_password')
    def test_get_issue_not_found(self, mock_get_password):
        """Test issue not found error"""
        mock_get_password.return_value = 'test-api-token'
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 404
            mock_get.return_value = mock_response
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            
            with pytest.raises(ValueError, match="JIRA issue not found: PROJ-999"):
                client.get_issue('PROJ-999')

    @patch('keyring.get_password')
    def test_get_issue_invalid_key_format(self, mock_get_password):
        """Test invalid issue key format"""
        mock_get_password.return_value = 'test-api-token'
        
        client = JiraClient('https://test.atlassian.net', 'test@example.com')
        
        # These should all fail validation before making API calls
        truly_invalid_keys = ['invalid', '123', 'PROJ', 'PROJ-']
        
        for invalid_key in truly_invalid_keys:
            with pytest.raises(ValueError, match="Invalid JIRA issue key format"):
                client.get_issue(invalid_key)
        
        # Test that 'proj-123' gets normalized to 'PROJ-123' and then fails with 404
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.status_code = 404
            mock_get.return_value = mock_response
            
            with pytest.raises(ValueError, match="JIRA issue not found: PROJ-123"):
                client.get_issue('proj-123')

    @patch('keyring.get_password')
    def test_get_issue_key_normalization(self, mock_get_password):
        """Test issue key normalization"""
        mock_get_password.return_value = 'test-api-token'
        
        mock_response_data = {
            'key': 'PROJ-123',
            'fields': {'summary': 'Test Issue'}
        }
        
        with patch.object(requests.Session, 'get') as mock_get:
            mock_response = Mock()
            mock_response.raise_for_status.return_value = None
            mock_response.json.return_value = mock_response_data
            mock_get.return_value = mock_response
            
            client = JiraClient('https://test.atlassian.net', 'test@example.com')
            
            # Test that lowercase and whitespace are handled
            issue = client.get_issue(' proj-123 ')
            assert issue.key == 'PROJ-123'
            
            # Verify URL was called with uppercase key
            args, kwargs = mock_get.call_args
            assert '/rest/api/3/issue/PROJ-123' in args[0]
