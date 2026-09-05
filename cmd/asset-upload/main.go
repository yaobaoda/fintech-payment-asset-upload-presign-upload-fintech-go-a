package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/fintech-asset-upload/internal/infrai"
	"example.com/fintech-asset-upload/internal/uploadpolicy"
)

const bucketName = "fintech-payment-assets"

type server struct {
	storage *infrai.Client
	now     func() time.Time
}

type authorizationResponse struct {
	uploadpolicy.Decision
	UploadURL string    `json:"upload_url,omitempty"`
	Method    string    `json:"method,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	client := infrai.NewClient(apiKey)
	s := &server{storage: client, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /upload-authorizations", s.authorizeUpload)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	addr := ":8080"
	log.Printf("asset authorization service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) authorizeUpload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var input uploadpolicy.Request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	decision := uploadpolicy.Authorize(input)
	now := s.now().UTC()
	response := authorizationResponse{Decision: decision, DecidedAt: now}
	if decision.Action != "issue_upload_url" {
		writeJSON(w, http.StatusOK, response)
		return
	}

	const lifetime = 5 * time.Minute
	result, err := s.storage.PresignPut(r.Context(), bucketName, decision.ObjectKey, infrai.PresignRequest{
		Op: "put", ExpiresSeconds: int(lifetime.Seconds()), ContentType: input.ContentType,
		MaxBytes: input.SizeBytes, IdempotencyKey: decision.ID,
	})
	if err != nil {
		var apiErr *infrai.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
			writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Code, "message": apiErr.Message})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "storage request failed"})
		return
	}

	response.UploadURL = result.URL
	response.Method = http.MethodPut
	response.ExpiresAt = now.Add(lifetime)
	writeJSON(w, http.StatusCreated, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
