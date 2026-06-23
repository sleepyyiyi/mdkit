package markdown

import "testing"

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"http allowed", "http://example.com", "http://example.com"},
		{"https allowed", "https://example.com/a?b=1", "https://example.com/a?b=1"},
		{"mailto allowed", "mailto:a@b.com", "mailto:a@b.com"},
		{"relative path allowed", "/docs/page", "/docs/page"},
		{"anchor allowed", "#section", "#section"},
		{"relative file allowed", "page.html", "page.html"},
		{"javascript dropped", "javascript:alert(1)", ""},
		{"JavaScript mixed-case dropped", "JaVaScRiPt:alert(1)", ""},
		{"data uri dropped", "data:text/html,<script>", ""},
		{"vbscript dropped", "vbscript:msgbox(1)", ""},
		{"file scheme dropped", "file:///etc/passwd", ""},
		{"empty dropped", "", ""},
		{"whitespace dropped", "   ", ""},
		{"leading space trimmed then allowed", "  https://x.com", "https://x.com"},
		{"tab-obfuscated scheme dropped", "java\tscript:alert(1)", ""},
		{"path-with-colon treated relative", "path/to:thing", "path/to:thing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeURL(tt.in); got != tt.want {
				t.Fatalf("sanitizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
