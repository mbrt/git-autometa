"""
JIRA to Markdown converter for git-autometa
Converts JIRA markup syntax to GitHub-compatible Markdown
"""

import subprocess
from typing import Optional


def convert_jira_to_markdown(jira_text: Optional[str]) -> str:
    """
    Convenience function to convert JIRA markup to Markdown

    Args:
        jira_text: JIRA markup text to convert

    Returns:
        Converted Markdown text
    """
    if not jira_text:
        return ""

    result = subprocess.run(
        [
            "pandoc",
            "-f", "jira",
            "-t", "gfm"
        ],
        input=jira_text.encode(),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True
    )
    # Remove extra whitespace, but add it back, if the original had it
    return result.stdout.decode()
