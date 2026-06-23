package markdown

import (
	"strings"
	"testing"
)

func TestParse_BasicFormatting(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		contains string
	}{
		{"h1", "# Title", "<h1>Title</h1>"},
		{"h3", "### Sub", "<h3>Sub</h3>"},
		{"bold", "**strong**", "<strong>strong</strong>"},
		{"italic", "*em*", "<em>em</em>"},
		{"inline code", "`x`", "<code>x</code>"},
		{"paragraph", "hello world", "<p>hello world</p>"},
		{"unordered list", "- a\n- b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>"},
		{"ordered list", "1. a\n2. b", "<ol>\n<li>a</li>\n<li>b</li>\n</ol>"},
		{"blockquote", "> quote", "<blockquote>quote</blockquote>"},
		{"safe link", "[go](https://go.dev)", `<a href="https://go.dev" rel="noopener noreferrer">go</a>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.in)
			if !strings.Contains(got, tt.contains) {
				t.Fatalf("Parse(%q) = %q, want contains %q", tt.in, got, tt.contains)
			}
		})
	}
}

// TestParse_XSS is the security core: untrusted Markdown must never produce
// executable HTML.
func TestParse_XSS(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		mustNotHave string
		mustHave    string
	}{
		{
			name:        "raw script tag escaped",
			in:          "<script>alert(1)</script>",
			mustNotHave: "<script>",
			mustHave:    "&lt;script&gt;",
		},
		{
			name:        "img onerror escaped",
			in:          `<img src=x onerror=alert(1)>`,
			mustNotHave: "<img",
			mustHave:    "&lt;img",
		},
		{
			name:        "javascript link neutralized",
			in:          "[click](javascript:alert(1))",
			mustNotHave: "javascript:",
			mustHave:    "click",
		},
		{
			name:        "data uri link neutralized",
			in:          "[x](data:text/html,<script>alert(1)</script>)",
			mustNotHave: "data:text/html",
			mustHave:    "x",
		},
		{
			name:        "html in code fence escaped",
			in:          "```\n<script>alert(1)</script>\n```",
			mustNotHave: "<script>",
			mustHave:    "&lt;script&gt;",
		},
		{
			name:        "attribute breakout escaped",
			in:          `[x](https://a.com" onmouseover="alert(1))`,
			mustNotHave: `onmouseover="alert`,
			mustHave:    "x",
		},
		{
			name:        "svg onload escaped",
			in:          `<svg onload=alert(1)>`,
			mustNotHave: "<svg",
			mustHave:    "&lt;svg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.in)
			if strings.Contains(got, tt.mustNotHave) {
				t.Fatalf("Parse(%q) = %q, must NOT contain %q", tt.in, got, tt.mustNotHave)
			}
			if tt.mustHave != "" && !strings.Contains(got, tt.mustHave) {
				t.Fatalf("Parse(%q) = %q, must contain %q", tt.in, got, tt.mustHave)
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"only whitespace", "   \n  \n"},
		{"unclosed code fence", "```\ncode without close"},
		{"unclosed bold", "**not closed"},
		{"empty link", "[]()"},
		{"nested-ish markers", "***bolditalic***"},
		{"many open brackets", strings.Repeat("[", 1000)},
		{"many asterisks", strings.Repeat("*", 1000)},
		{"deeply nested list markers", strings.Repeat("- ", 500)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Must not panic and must always return escaped (no raw < >).
			got := Parse(tt.in)
			if strings.Contains(got, "<script") {
				t.Fatalf("unexpected raw script in output: %q", got)
			}
		})
	}
}

func TestParse_UnclosedFenceNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parse panicked on unclosed fence: %v", r)
		}
	}()
	out := Parse("```go\nfmt.Println(\"hi\")")
	if !strings.Contains(out, "<pre><code>") {
		t.Fatalf("expected code block, got %q", out)
	}
}

// BenchmarkParse_Pathological guards against quadratic / catastrophic behavior
// on adversarial input. Go's RE2 regexp is linear, so this should stay fast.
func BenchmarkParse_Pathological(b *testing.B) {
	input := strings.Repeat("[a](*b*`c`**d**)", 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Parse(input)
	}
}
