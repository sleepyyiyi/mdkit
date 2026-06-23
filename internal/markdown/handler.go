package markdown

import (
	"encoding/json"
	"net/http"
)

// RegisterRoutes mounts the markdown endpoints onto mux. Handlers do JSON
// encode/decode only — all logic lives in Service.
func RegisterRoutes(mux *http.ServeMux, svc *Service) {
	mux.HandleFunc("POST /convert", handleConvert(svc))
	mux.HandleFunc("POST /summarize", handleSummarize(svc))
}

func handleConvert(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Defense in depth: cap the body at the transport boundary.
		r.Body = http.MaxBytesReader(w, r.Body, MaxInputBytes+1024)

		var req ConvertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		resp, err := svc.Convert(req.Markdown)
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "input too large")
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleSummarize(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxInputBytes+1024)

		var req SummarizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		resp, err := svc.Summarize(r.Context(), req.Markdown, req.MaxWords)
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "input too large")
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Response already partially written; log-only in real systems.
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// msg is a fixed internal constant, never user input — no injection risk.
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
