package markdown

import (
	"strings"
	"testing"
)

func TestConvertJiraToMarkdown_Headings(t *testing.T) {
	in := "h1. Title\nMore\nh3. Small\n"
	want := "# Title\nMore\n### Small\n"
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("headings:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_Lists(t *testing.T) {
	in := strings.Join([]string{
		"* Top bullet",
		"** Nested bullet",
		"# Top ordered",
		"## Nested ordered",
		"Not a list: *inline* here",
		"",
	}, "\n")
	want := strings.Join([]string{
		"- Top bullet",
		"  - Nested bullet",
		"1. Top ordered",
		"  1. Nested ordered",
		"Not a list: **inline** here",
		"",
	}, "\n")
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("lists:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_InlineStylesAndLinks(t *testing.T) {
	in := strings.Join([]string{
		"This is *bold* and _italic_ and -strike- and +under+.",
		"Code inline: {{x := 1}}.",
		"Link classic: [Example|https://example.com]",
		"Link bare: [https://example.org]",
		"",
	}, "\n")
	want := strings.Join([]string{
		"This is **bold** and *italic* and ~~strike~~ and <u>under</u>.",
		"Code inline: `x := 1`.",
		"Link classic: [Example](https://example.com)",
		"Link bare: https://example.org",
		"",
	}, "\n")
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("inline styles/links:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_CodeBlocks(t *testing.T) {
	in := strings.Join([]string{
		"Before",
		"{code:go}",
		"fmt.Println(\"hi\")",
		"{code}",
		"After",
		"",
	}, "\n")
	want := strings.Join([]string{
		"Before",
		"```go",
		"fmt.Println(\"hi\")",
		"```",
		"After",
		"",
	}, "\n")
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("code blocks:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_QuoteBlocks(t *testing.T) {
	in := strings.Join([]string{
		"{quote}",
		"Line 1",
		"Line 2",
		"{quote}",
		"",
	}, "\n")
	want := strings.Join([]string{
		"> Line 1",
		"> Line 2",
		"",
	}, "\n")
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("quote blocks:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_Tables_HeaderAndBody(t *testing.T) {
	in := strings.Join([]string{
		"|| H1 || H2 ||",
		"| c1 | c2 |",
		"| c3 | c4 |",
		"",
	}, "\n")
	want := strings.Join([]string{
		"| H1 | H2 |",
		"| --- | --- |",
		"| c1 | c2 |",
		"| c3 | c4 |",
		"",
	}, "\n")
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("tables:\nwant:\n%q\n got:\n%q", want, got)
	}
}

func TestConvertJiraToMarkdown_NewlineNormalization(t *testing.T) {
	in := "h2. Title\r\n* item\r\n"
	want := "## Title\n- item\n"
	got := ConvertJiraToMarkdown(in)
	if got != want {
		t.Fatalf("newlines:\nwant:\n%q\n got:\n%q", want, got)
	}
}
