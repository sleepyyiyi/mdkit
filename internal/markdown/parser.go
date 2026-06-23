package markdown

import (
	"html"
	"regexp"
	"strings"
)

// Inline patterns operate on already-HTML-escaped text.
//
// IMPORTANT: Go's regexp package is RE2-based and runs in linear time — it has
// no catastrophic backtracking, so classic ReDoS (as seen with PCRE `.*` nesting)
// is not possible here. We still bound input size in the service layer to cap
// memory/CPU on huge documents. See QA_REPORT.md §performance.
var (
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reBold       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reItalic     = regexp.MustCompile(`\*([^*]+)\*`)
	reLink       = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	reHeading    = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reOrdered    = regexp.MustCompile(`^\s*(\d+)\.\s+(.*)$`)
)

// Parse converts a documented subset of Markdown into sanitized HTML.
//
// Security invariant: every piece of text is HTML-escaped BEFORE any inline
// formatting is applied, so raw HTML in the input (e.g. <script>) is rendered
// as literal text and never executed. Link URLs are checked by sanitizeURL.
func Parse(src string) string {
	var b strings.Builder
	lines := strings.Split(src, "\n")

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "```"):
			i = writeCodeBlock(&b, lines, i)

		case reHeading.MatchString(line):
			writeHeading(&b, line)
			i++

		case strings.HasPrefix(trimmed, ">"):
			i = writeBlockquote(&b, lines, i)

		case isUnorderedItem(line):
			i = writeUnorderedList(&b, lines, i)

		case reOrdered.MatchString(line):
			i = writeOrderedList(&b, lines, i)

		case trimmed == "":
			i++

		default:
			i = writeParagraph(&b, lines, i)
		}
	}

	return b.String()
}

func writeCodeBlock(b *strings.Builder, lines []string, i int) int {
	i++ // consume opening fence
	var code strings.Builder
	for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
		code.WriteString(html.EscapeString(lines[i]))
		code.WriteByte('\n')
		i++
	}
	if i < len(lines) {
		i++ // consume closing fence (unclosed fence is tolerated: no panic)
	}
	b.WriteString("<pre><code>")
	b.WriteString(code.String())
	b.WriteString("</code></pre>\n")
	return i
}

func writeHeading(b *strings.Builder, line string) {
	m := reHeading.FindStringSubmatch(line)
	level := byte('0' + len(m[1]))
	b.WriteString("<h")
	b.WriteByte(level)
	b.WriteByte('>')
	b.WriteString(inline(m[2]))
	b.WriteString("</h")
	b.WriteByte(level)
	b.WriteString(">\n")
}

func writeBlockquote(b *strings.Builder, lines []string, i int) int {
	var quote []string
	for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
		q := strings.TrimSpace(lines[i])
		quote = append(quote, strings.TrimSpace(strings.TrimPrefix(q, ">")))
		i++
	}
	b.WriteString("<blockquote>")
	b.WriteString(inline(strings.Join(quote, " ")))
	b.WriteString("</blockquote>\n")
	return i
}

func writeUnorderedList(b *strings.Builder, lines []string, i int) int {
	b.WriteString("<ul>\n")
	for i < len(lines) && isUnorderedItem(lines[i]) {
		item := strings.TrimSpace(lines[i])[2:]
		b.WriteString("<li>")
		b.WriteString(inline(item))
		b.WriteString("</li>\n")
		i++
	}
	b.WriteString("</ul>\n")
	return i
}

func writeOrderedList(b *strings.Builder, lines []string, i int) int {
	b.WriteString("<ol>\n")
	for i < len(lines) {
		m := reOrdered.FindStringSubmatch(lines[i])
		if m == nil {
			break
		}
		b.WriteString("<li>")
		b.WriteString(inline(m[2]))
		b.WriteString("</li>\n")
		i++
	}
	b.WriteString("</ol>\n")
	return i
}

func writeParagraph(b *strings.Builder, lines []string, i int) int {
	var para []string
	for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !isBlockStart(lines[i]) {
		para = append(para, strings.TrimSpace(lines[i]))
		i++
	}
	b.WriteString("<p>")
	b.WriteString(inline(strings.Join(para, " ")))
	b.WriteString("</p>\n")
	return i
}

// inline escapes the text, then applies inline formatting on the escaped form.
func inline(text string) string {
	esc := html.EscapeString(text)
	esc = reInlineCode.ReplaceAllString(esc, "<code>$1</code>")
	esc = reBold.ReplaceAllString(esc, "<strong>$1</strong>")
	esc = reItalic.ReplaceAllString(esc, "<em>$1</em>")
	esc = reLink.ReplaceAllStringFunc(esc, renderLink)
	return esc
}

// renderLink builds an <a> tag with a scheme-checked URL. If the URL is unsafe
// (e.g. javascript:), the link is neutralized to its plain text label.
func renderLink(match string) string {
	m := reLink.FindStringSubmatch(match)
	label, rawURL := m[1], m[2]
	safeURL := sanitizeURL(rawURL)
	if safeURL == "" {
		return label
	}
	return `<a href="` + safeURL + `" rel="noopener noreferrer">` + label + `</a>`
}

func isUnorderedItem(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ ")
}

func isBlockStart(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return true
	}
	if strings.HasPrefix(t, "#") || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "```") {
		return true
	}
	return isUnorderedItem(line) || reOrdered.MatchString(line)
}
