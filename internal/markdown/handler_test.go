package markdown

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMux() *http.ServeMux {
	svc := NewService(NewMockLLM())
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)
	return mux
}

func TestHandleConvert_Normal(t *testing.T) {
	mux := setupMux()
	body := `{"markdown":"# Hi"}`
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp ConvertResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(resp.HTML, "<h1>Hi</h1>") {
		t.Fatalf("expected heading, got %q", resp.HTML)
	}
}

func TestHandleConvert_XSSNeutralized(t *testing.T) {
	mux := setupMux()
	payload, _ := json.Marshal(ConvertRequest{Markdown: "<script>alert(1)</script>"})
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp ConvertResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if strings.Contains(resp.HTML, "<script>") {
		t.Fatalf("script tag leaked into output: %q", resp.HTML)
	}
}

func TestHandleConvert_InvalidJSON(t *testing.T) {
	mux := setupMux()
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSummarize_Normal(t *testing.T) {
	mux := setupMux()
	body := `{"markdown":"The quick brown fox.","max_words":2}`
	req := httptest.NewRequest(http.MethodPost, "/summarize", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp SummarizeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.AIAvailable {
		t.Fatal("expected ai_available=true")
	}
}

func TestHandleSummarize_InvalidJSON(t *testing.T) {
	mux := setupMux()
	req := httptest.NewRequest(http.MethodPost, "/summarize", bytes.NewBufferString("nope"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleConvert_OversizedRejected(t *testing.T) {
	mux := setupMux()
	huge := strings.Repeat("a", MaxInputBytes+2048)
	payload, _ := json.Marshal(ConvertRequest{Markdown: huge})
	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	// Either 400 (MaxBytesReader trips decode) or 413 (service rejects) is acceptable;
	// both prove the input was not processed.
	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 400 or 413 for oversized input, got %d", w.Code)
	}
}
