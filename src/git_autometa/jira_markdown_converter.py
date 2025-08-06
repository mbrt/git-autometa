"""
JIRA to Markdown converter for git-autometa
Converts JIRA markup syntax to GitHub-compatible Markdown
"""

import re
from typing import Optional


class JiraToMarkdownConverter:
    """Converts JIRA markup to Markdown format"""

    def __init__(self):
        """Initialize the converter with regex patterns"""
        # Define conversion patterns
        self.patterns = [
            # Links: [text|url] -> [text](url)
            (r'\[([^\|\]]+)\|([^\]]+)\]', r'[\1](\2)'),
            
            # Bold: *text* -> **text** (but avoid conflicts with list markers)
            (r'(?<!\*)\*([^*\n]+?)\*(?!\*)', r'**\1**'),
            
            # Italic: _text_ -> *text*
            (r'_([^_\n]+?)_', r'*\1*'),
            
            # Code spans: {{text}} -> `text`
            (r'\{\{([^}]+?)\}\}', r'`\1`'),
            
            # Strikethrough: -text- -> ~~text~~
            (r'(?<!-)-([^-\n]+?)-(?!-)', r'~~\1~~'),
            
            # Underline: +text+ -> **text** (Markdown doesn't have underline, use bold)
            (r'\+([^+\n]+?)\+', r'**\1**'),
            
            # Superscript: ^text^ -> <sup>text</sup>
            (r'\^([^^]+?)\^', r'<sup>\1</sup>'),
            
            # Subscript: ~text~ -> <sub>text</sub>
            (r'~([^~]+?)~', r'<sub>\1</sub>'),
        ]

    def convert_headers(self, text: str) -> str:
        """Convert JIRA headers to Markdown headers"""
        # Headers: h1. Title -> # Title, h2. -> ##, etc.
        def header_replacer(match):
            level = int(match.group(1))
            title = match.group(2).strip()
            return '#' * level + ' ' + title
        
        return re.sub(r'^h([1-6])\.\s*(.+)$', header_replacer, text, flags=re.MULTILINE)

    def convert_lists(self, text: str) -> str:
        """Convert JIRA lists to Markdown lists"""
        lines = text.split('\n')
        converted_lines = []
        
        for line in lines:
            # Skip lines that are original JIRA headers (not yet processed)
            if re.match(r'^h[1-6]\.\s', line):
                converted_lines.append(line)
                continue
                
            # Numbered lists: # item -> 1. item (JIRA numbered lists)
            numbered_match = re.match(r'^(#+)\s+(.+)$', line)
            if numbered_match:
                level = len(numbered_match.group(1))
                content = numbered_match.group(2)
                indent = '  ' * (level - 1)
                converted_lines.append(f"{indent}1. {content}")
            # Bullet lists with multiple levels: ** item -> * item (with proper indentation)
            else:
                bullet_match = re.match(r'^(\*+)\s+(.+)$', line)
                if bullet_match:
                    level = len(bullet_match.group(1))
                    content = bullet_match.group(2)
                    indent = '  ' * (level - 1)
                    converted_lines.append(f"{indent}* {content}")
                else:
                    converted_lines.append(line)
        
        return '\n'.join(converted_lines)

    def convert_code_blocks(self, text: str) -> str:
        """Convert JIRA code blocks to Markdown code blocks"""
        # Code blocks: {code} ... {code} -> ``` ... ```
        # Also handle language-specific blocks: {code:java} ... {code}
        
        # Language-specific code blocks
        text = re.sub(
            r'\{code:([^}]+)\}(.*?)\{code\}',
            r'```\1\2```',
            text,
            flags=re.DOTALL
        )
        
        # Generic code blocks
        text = re.sub(
            r'\{code\}(.*?)\{code\}',
            r'```\1```',
            text,
            flags=re.DOTALL
        )
        
        return text

    def convert_quotes(self, text: str) -> str:
        """Convert JIRA quotes to Markdown blockquotes"""
        # Quote blocks: {quote} ... {quote} -> > ...
        def quote_replacer(match):
            quote_content = match.group(1).strip()
            # Add > to each line
            quoted_lines = []
            for line in quote_content.split('\n'):
                quoted_lines.append('> ' + line if line.strip() else '>')
            return '\n'.join(quoted_lines)
        
        return re.sub(
            r'\{quote\}(.*?)\{quote\}',
            quote_replacer,
            text,
            flags=re.DOTALL
        )

    def convert_noformat(self, text: str) -> str:
        """Convert JIRA noformat blocks to Markdown code blocks"""
        # No format blocks: {noformat} ... {noformat} -> ``` ... ```
        return re.sub(
            r'\{noformat\}(.*?)\{noformat\}',
            r'```\1```',
            text,
            flags=re.DOTALL
        )

    def convert_tables(self, text: str) -> str:
        """Convert JIRA tables to Markdown tables"""
        lines = text.split('\n')
        converted_lines = []
        in_table = False
        
        for i, line in enumerate(lines):
            # JIRA table rows start and end with ||
            if line.strip().startswith('||') and line.strip().endswith('||'):
                if not in_table:
                    in_table = True
                    # This is a header row
                    cells = [cell.strip() for cell in line.strip('|').split('||')]
                    converted_lines.append('| ' + ' | '.join(cells) + ' |')
                    # Add separator row
                    converted_lines.append('| ' + ' | '.join(['---'] * len(cells)) + ' |')
                else:
                    # Regular table row
                    cells = [cell.strip() for cell in line.strip('|').split('||')]
                    converted_lines.append('| ' + ' | '.join(cells) + ' |')
            elif line.strip().startswith('|') and line.strip().endswith('|') and in_table:
                # Regular table row (alternative format)
                cells = [cell.strip() for cell in line.strip('|').split('|')]
                converted_lines.append('| ' + ' | '.join(cells) + ' |')
            else:
                if in_table:
                    in_table = False
                converted_lines.append(line)
        
        return '\n'.join(converted_lines)

    def clean_whitespace(self, text: str) -> str:
        """Clean up whitespace issues common in JIRA content"""
        # Remove excessive blank lines
        text = re.sub(r'\n\s*\n\s*\n', '\n\n', text)
        
        # Trim trailing whitespace from lines
        lines = [line.rstrip() for line in text.split('\n')]
        
        return '\n'.join(lines).strip()

    def convert(self, jira_text: Optional[str]) -> str:
        """
        Convert JIRA markup text to Markdown
        
        Args:
            jira_text: JIRA markup text to convert
            
        Returns:
            Converted Markdown text
        """
        if not jira_text:
            return ""
        
        text = jira_text
        
        # Apply conversions in order
        text = self.convert_code_blocks(text)
        text = self.convert_noformat(text)
        text = self.convert_quotes(text)
        text = self.convert_tables(text)
        
        # Apply regex patterns first (before lists to avoid conflicts)
        for pattern, replacement in self.patterns:
            if callable(replacement):
                text = re.sub(pattern, replacement, text, flags=re.MULTILINE)
            else:
                text = re.sub(pattern, replacement, text, flags=re.MULTILINE)
        
        text = self.convert_lists(text)
        text = self.convert_headers(text)
        
        # Final cleanup
        text = self.clean_whitespace(text)
        
        return text


# Global converter instance
_converter = JiraToMarkdownConverter()


def convert_jira_to_markdown(jira_text: Optional[str]) -> str:
    """
    Convenience function to convert JIRA markup to Markdown
    
    Args:
        jira_text: JIRA markup text to convert
        
    Returns:
        Converted Markdown text
    """
    return _converter.convert(jira_text)
