"""
Tests for JIRA to Markdown converter
"""

import pytest
from src.git_autometa.jira_markdown_converter import convert_jira_to_markdown


class TestJiraMarkdownConverter:
    """Test cases for JIRA to Markdown conversion"""

    def test_jira_links(self):
        """Test JIRA link conversion"""
        input_text = "Check out [this documentation|https://example.com/docs] for more info."
        expected = "Check out [this documentation](https://example.com/docs) for more info."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_bold_text(self):
        """Test bold text conversion"""
        input_text = "This is *bold text* in JIRA."
        expected = "This is **bold text** in JIRA."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_italic_text(self):
        """Test italic text conversion"""
        input_text = "This is _italic text_ in JIRA."
        expected = "This is *italic text* in JIRA."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_code_spans(self):
        """Test code span conversion"""
        input_text = "Use {{console.log()}} to debug."
        expected = "Use `console.log()` to debug."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_headers(self):
        """Test header conversion"""
        input_text = "h1. Main Title\nh2. Subtitle\nh3. Section"
        expected = "# Main Title\n## Subtitle\n### Section"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_code_blocks(self):
        """Test code block conversion"""
        input_text = "{code:javascript}\nfunction hello() {\n  console.log('Hello');\n}\n{code}"
        expected = "```javascript\nfunction hello() {\n  console.log('Hello');\n}\n```"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_bullet_lists(self):
        """Test bullet list conversion"""
        input_text = "* Item 1\n** Sub item 1\n** Sub item 2\n* Item 2"
        expected = "* Item 1\n  * Sub item 1\n  * Sub item 2\n* Item 2"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_numbered_lists(self):
        """Test numbered list conversion"""
        input_text = "# First item\n## Sub item\n# Second item"
        expected = "1. First item\n  1. Sub item\n1. Second item"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_quote_blocks(self):
        """Test quote block conversion"""
        input_text = "{quote}\nThis is a quote\nwith multiple lines\n{quote}"
        expected = "> This is a quote\n> with multiple lines"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_empty_input(self):
        """Test empty input handling"""
        assert convert_jira_to_markdown("") == ""
        assert convert_jira_to_markdown(None) == ""

    def test_no_formatting(self):
        """Test plain text without formatting"""
        input_text = "This is plain text with no JIRA formatting."
        expected = "This is plain text with no JIRA formatting."
        result = convert_jira_to_markdown(input_text)
        assert result == expected


    def test_underline_to_bold(self):
        """Test underline to bold conversion (Markdown doesn't support underline)"""
        input_text = "This is +underlined+ text."
        expected = "This is **underlined** text."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_superscript(self):
        """Test superscript conversion"""
        input_text = "E = mc^2^"
        expected = "E = mc<sup>2</sup>"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_subscript(self):
        """Test subscript conversion"""
        input_text = "H~2~O"
        expected = "H<sub>2</sub>O"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_generic_code_block(self):
        """Test generic code block without language"""
        input_text = "{code}\nfunction example() {\n  return true;\n}\n{code}"
        expected = "```\nfunction example() {\n  return true;\n}\n```"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_noformat_block(self):
        """Test noformat block conversion"""
        input_text = "{noformat}\nThis is preformatted text\nwith preserved spacing\n{noformat}"
        expected = "```\nThis is preformatted text\nwith preserved spacing\n```"
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_multiple_links(self):
        """Test multiple links in text"""
        input_text = "See [docs|http://docs.com] and [API|http://api.com] for info."
        expected = "See [docs](http://docs.com) and [API](http://api.com) for info."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_mixed_text_formatting(self):
        """Test mixed text formatting"""
        input_text = "This has *bold* and _italic_ and {{code}} formatting."
        expected = "This has **bold** and *italic* and `code` formatting."
        result = convert_jira_to_markdown(input_text)
        assert result == expected

    def test_header_levels(self):
        """Test all header levels"""
        input_text = "h1. Header 1\nh2. Header 2\nh3. Header 3\nh4. Header 4\nh5. Header 5\nh6. Header 6"
        expected = "# Header 1\n## Header 2\n### Header 3\n#### Header 4\n##### Header 5\n###### Header 6"
        result = convert_jira_to_markdown(input_text)
        assert result == expected
