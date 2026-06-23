// Package markdown converts untrusted Markdown into sanitized (XSS-safe) HTML,
// and provides an optional LLM-backed document summary with prompt-injection
// defense and graceful degradation.
//
// Scope note: the parser supports a documented SUBSET of Markdown (headings,
// emphasis, inline code, fenced code blocks, links, unordered/ordered lists,
// blockquotes, paragraphs). The security-critical work is in sanitizer.go:
// raw HTML in the input is always escaped, link URLs are scheme-checked, and
// no event-handler attributes are ever emitted.
package markdown

// ConvertRequest is the JSON body for POST /convert.
type ConvertRequest struct {
	Markdown string `json:"markdown"`
}

// ConvertResponse is the JSON body returned by POST /convert.
type ConvertResponse struct {
	HTML  string `json:"html"`
	Bytes int    `json:"bytes"`
}

// SummarizeRequest is the JSON body for POST /summarize.
type SummarizeRequest struct {
	Markdown string `json:"markdown"`
	// MaxWords optionally caps the summary length; 0 means service default.
	MaxWords int `json:"max_words,omitempty"`
}

// SummarizeResponse is the JSON body returned by POST /summarize.
type SummarizeResponse struct {
	Summary     string `json:"summary"`
	AIAvailable bool   `json:"ai_available"`
}

// MaxInputBytes bounds request payloads to prevent resource exhaustion / ReDoS
// amplification. Inputs larger than this are rejected before parsing.
const MaxInputBytes = 1 << 20 // 1 MiB
