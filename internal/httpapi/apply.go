package httpapi

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func (h *Handlers) apply(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 10 << 20 // 10 MB
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		writeError(w, http.StatusBadRequest, "empty request body")
		return
	}

	target := os.Getenv("NEXTJS_APPLY_URL")
	if target == "" {
		log.Printf("apply: NEXTJS_APPLY_URL is not set")
		writeError(w, http.StatusInternalServerError, "apply endpoint not configured")
		return
	}

	req, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		target,
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("apply: build upstream request failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to build upstream request")
		return
	}

	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("apply: upstream request failed: %v", err)
		writeError(w, http.StatusBadGateway, "upstream apply request failed")
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("apply: read upstream body failed: %v", err)
		writeError(w, http.StatusBadGateway, "failed to read upstream response")
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}
