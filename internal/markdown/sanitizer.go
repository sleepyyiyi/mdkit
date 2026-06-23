package markdown

import "strings"

// allowedSchemes is the allowlist of URL schemes permitted in generated links.
// Everything else (javascript:, data:, vbscript:, file:, …) is rejected.
var allowedSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
}

// sanitizeURL returns the URL when it is safe to emit in an href, or "" to
// signal the caller to neutralize the link (render as plain text).
//
// Rules:
//   - empty / whitespace-only            → "" (drop)
//   - no scheme (relative URL, #anchor)  → allowed
//   - scheme in allowlist                → allowed
//   - any other scheme                   → "" (drop)
//
// The scheme is matched case-insensitively. Obfuscation via control characters
// or whitespace inside the scheme fails the allowlist lookup and is dropped.
func sanitizeURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	colon := strings.IndexByte(trimmed, ':')
	if colon == -1 {
		return trimmed // relative URL, no scheme
	}

	// A path/query/fragment before the first ':' means it isn't a scheme,
	// e.g. "path/to:thing" or "#a:b" — treat as a relative URL.
	scheme := trimmed[:colon]
	if strings.ContainsAny(scheme, "/?#") {
		return trimmed
	}

	if allowedSchemes[strings.ToLower(scheme)] {
		return trimmed
	}
	return ""
}
