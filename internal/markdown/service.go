package markdown

import (
	"context"
	"fmt"
	"strings"
)

const defaultSummaryWords = 40

// Service orchestrates Markdown conversion and summarization. It owns no HTTP
// concerns — that is the handler's job.
type Service struct {
	llm LLM
}

// NewService wires up the markdown service with an LLM backend for summaries.
func NewService(llm LLM) *Service {
	return &Service{llm: llm}
}

// Convert parses Markdown into sanitized HTML, rejecting oversized input.
func (s *Service) Convert(src string) (*ConvertResponse, error) {
	if len(src) > MaxInputBytes {
		return nil, fmt.Errorf("input %d exceeds limit %d bytes", len(src), MaxInputBytes)
	}
	out := Parse(src)
	return &ConvertResponse{HTML: out, Bytes: len(out)}, nil
}

// Summarize produces a document summary via the LLM, falling back to a local
// extractive summary when the LLM fails or the context is cancelled.
func (s *Service) Summarize(ctx context.Context, src string, maxWords int) (*SummarizeResponse, error) {
	if len(src) > MaxInputBytes {
		return nil, fmt.Errorf("input %d exceeds limit %d bytes", len(src), MaxInputBytes)
	}
	if maxWords <= 0 {
		maxWords = defaultSummaryWords
	}

	plain := toPlainText(src)
	summary, err := s.llm.Summarize(ctx, plain, maxWords)
	if err != nil {
		return &SummarizeResponse{
			Summary:     extractiveFallback(plain, maxWords),
			AIAvailable: false,
		}, nil
	}
	return &SummarizeResponse{Summary: summary, AIAvailable: true}, nil
}

// toPlainText strips Markdown markers so the LLM receives clean prose. It is
// also a defense layer: structural markup is removed before the text reaches
// the model.
func toPlainText(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	inFence := false

	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, t)
			continue
		}
		if m := reOrdered.FindStringSubmatch(t); m != nil {
			t = m[2]
		} else {
			t = strings.TrimLeft(t, "#>-*+ \t")
		}
		out = append(out, t)
	}

	joined := strings.Join(out, " ")
	joined = strings.ReplaceAll(joined, "**", "")
	joined = strings.ReplaceAll(joined, "`", "")
	return strings.Join(strings.Fields(joined), " ")
}

// extractiveFallback returns the first maxWords words of plain text.
func extractiveFallback(plain string, maxWords int) string {
	words := strings.Fields(plain)
	if len(words) > maxWords {
		return strings.Join(words[:maxWords], " ") + " …"
	}
	return strings.Join(words, " ")
}
