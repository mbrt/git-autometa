package markdown

import (
	"regexp"
	"strings"
)

// ConvertJiraToMarkdown converts a subset of Jira wiki markup into
// GitHub-flavored Markdown. Supported features:
// - Headings: h1. .. h6.
// - Lists: * bullet lists, # ordered lists (with nesting)
// - Inline styles: *bold*, _italic_, -strike-, +underline+, {{code}}
// - Links: [text|url] and [url]
// - Code blocks: {code[:lang]} ... {code}
// - Quote blocks: {quote} ... {quote}
// - Tables: ||h1||h2|| and |c1|c2|
func ConvertJiraToMarkdown(text string) string {
	if text == "" {
		return ""
	}

	// Normalize newlines
	s := strings.ReplaceAll(text, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Convert code blocks first to avoid interfering with inline replacements
	s = convertCodeBlocks(s)
	// Convert quote blocks
	s = convertQuoteBlocks(s)
	// Convert lists before headings so resulting Markdown headings ("# ") are not mistaken for lists
	s = convertLists(s)
	// Convert headings
	s = convertHeadings(s)
	// Convert links
	s = convertLinks(s)
	// Convert inline code
	s = convertInlineCode(s)
	// Convert inline styles (bold, italics, strike, underline)
	s = convertInlineStyles(s)
	// Convert tables (line-oriented) after inline styles so table separators are not altered
	s = convertTables(s)

	// Trim trailing whitespace on lines for neatness
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}

func convertHeadings(s string) string {
	// Jira headings: h1. Title -> # Title
	re := regexp.MustCompile(`(?m)^h([1-6])\.\s*(.*)$`)
	return re.ReplaceAllStringFunc(s, func(line string) string {
		m := re.FindStringSubmatch(line)
		if len(m) != 3 {
			return line
		}
		level := m[1]
		title := m[2]
		n := int(level[0] - '0')
		return strings.Repeat("#", n) + " " + title
	})
}

func convertLists(s string) string {
	// Convert leading sequences of * or # into nested lists
	// "* item" -> "- item"; "# item" -> "1. item"
	re := regexp.MustCompile(`(?m)^(?:[ \t]*)([*#]+)\s+(.*)$`)
	return re.ReplaceAllStringFunc(s, func(line string) string {
		m := re.FindStringSubmatch(line)
		if len(m) != 3 {
			return line
		}
		marks := m[1]
		content := m[2]
		if marks == "" {
			return line
		}
		level := len(marks)
		bulletChar := marks[0]
		indent := strings.Repeat("  ", level-1)
		if bulletChar == '*' {
			return indent + "- " + content
		}
		// ordered list
		return indent + "1. " + content
	})
}

func convertLinks(s string) string {
	// [text|url] -> [text](url)
	re1 := regexp.MustCompile(`\[([^\]|]+)\|([^\]]+)\]`)
	s = re1.ReplaceAllString(s, "[$1]($2)")
	// [http://example.com] -> http://example.com
	re2 := regexp.MustCompile(`\[(https?://[^\]]+)\]`)
	s = re2.ReplaceAllString(s, "$1")
	return s
}

func convertInlineCode(s string) string {
	// {{code}} -> `code`
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllString(s, "`$1`")
}

func convertInlineStyles(s string) string {
	// Bold: *bold* -> **bold** (avoid list line starts handled in convertLists)
	bold := regexp.MustCompile(`(?m)(^|[^*])\*([^*\n]+)\*`)
	s = bold.ReplaceAllString(s, "$1**$2**")
	// Italic: _italic_ -> *italic*
	italic := regexp.MustCompile(`_([^_\n]+)_`)
	s = italic.ReplaceAllString(s, "*$1*")
	// Strike: -strike- -> ~~strike~~
	strike := regexp.MustCompile(`-([^\-\n]+)-`)
	s = strike.ReplaceAllString(s, "~~$1~~")
	// Underline: +text+ -> <u>text</u>
	underline := regexp.MustCompile(`\+([^+\n]+)\+`)
	s = underline.ReplaceAllString(s, "<u>$1</u>")
	return s
}

func convertCodeBlocks(s string) string {
	// {code[:lang]}\n...\n{code}
	re := regexp.MustCompile(`(?s)\{code(?::([^}\n]+))?\}\n?(.*?)\n?\{code\}`)
	for {
		loc := re.FindStringSubmatchIndex(s)
		if loc == nil {
			break
		}
		matches := re.FindStringSubmatch(s)
		lang := matches[1]
		body := matches[2]
		fenced := "```"
		if lang != "" {
			fenced += strings.TrimSpace(lang)
		}
		fenced += "\n" + strings.TrimRight(body, "\n") + "\n```"
		s = s[:loc[0]] + fenced + s[loc[1]:]
	}
	return s
}

func convertQuoteBlocks(s string) string {
	// {quote}\n...\n{quote}
	re := regexp.MustCompile(`(?s)\{quote\}\n?(.*?)\n?\{quote\}`)
	for {
		loc := re.FindStringSubmatchIndex(s)
		if loc == nil {
			break
		}
		matches := re.FindStringSubmatch(s)
		body := strings.Trim(matches[1], "\n")
		// Prefix each line with "> "
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = "> " + strings.TrimRight(lines[i], " ")
		}
		out := strings.Join(lines, "\n")
		s = s[:loc[0]] + out + s[loc[1]:]
	}
	return s
}

func convertTables(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") { // possible table block
			// collect contiguous table lines
			start := i
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				i++
			}
			block := lines[start:i]
			converted := convertTableBlock(block)
			out = append(out, converted...)
			continue
		}
		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
}

func convertTableBlock(block []string) []string {
	if len(block) == 0 {
		return nil
	}
	// Determine if first row is a header (uses ||)
	first := strings.TrimSpace(block[0])
	isHeader := strings.Contains(first, "||")

	rows := make([][]string, 0, len(block))
	header := []string{}
	if isHeader {
		header = parseJiraTableRow(block[0], true)
		block = block[1:]
	}
	for _, ln := range block {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		rows = append(rows, parseJiraTableRow(ln, false))
	}

	var out []string
	if isHeader && len(header) > 0 {
		out = append(out, "| "+strings.Join(header, " | ")+" |")
		sep := make([]string, len(header))
		for i := range sep {
			sep[i] = "---"
		}
		out = append(out, "| "+strings.Join(sep, " | ")+" |")
	}
	for _, row := range rows {
		out = append(out, "| "+strings.Join(row, " | ")+" |")
	}
	return out
}

func parseJiraTableRow(line string, header bool) []string {
	// Normalize and trim pipes
	trimmed := strings.TrimSpace(line)
	if header {
		// Convert header delimiter to single pipe for simpler parsing
		trimmed = strings.ReplaceAll(trimmed, "||", "|")
	}
	// Remove all leading/trailing pipe characters and spaces
	trimmed = strings.Trim(trimmed, "| ")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		c := strings.TrimSpace(p)
		// Collapse multiple spaces
		c = strings.Join(strings.Fields(c), " ")
		cells = append(cells, c)
	}
	return cells
}
