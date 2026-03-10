package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var flakyCounter atomic.Int64

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", logMiddleware(handleHealth))
	mux.HandleFunc("/api/v1/validate-user/", logMiddleware(handleValidateUser))
	mux.HandleFunc("/api/v1/check-eligibility", logMiddleware(handleCheckEligibility))
	mux.HandleFunc("/api/v1/premium-check", logMiddleware(handlePremiumCheck))
	mux.HandleFunc("/api/v1/score", logMiddleware(handleScore))
	mux.HandleFunc("/api/v1/slow", logMiddleware(handleSlow))
	mux.HandleFunc("/api/v1/flaky", logMiddleware(handleFlaky))

	addr := ":9999"
	log.Printf("Mock external API server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func logMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(rec, r)
		log.Printf("%s %s %d", r.Method, r.URL.Path, rec.statusCode)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// GET /api/v1/health
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// GET /api/v1/validate-user/:userId
func handleValidateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	// Parse userId from path: /api/v1/validate-user/{userId}
	userID := strings.TrimPrefix(r.URL.Path, "/api/v1/validate-user/")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing userId"})
		return
	}

	if strings.HasPrefix(userID, "valid-") {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid":  true,
			"userId": userID,
		})
		return
	}

	writeJSON(w, http.StatusForbidden, map[string]any{
		"valid": false,
		"error": "forbidden",
	})
}

// POST /api/v1/check-eligibility
func handleCheckEligibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var body struct {
		TenantID  string `json:"tenantId"`
		ProgramID string `json:"programId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.TenantID == "blocked" {
		writeJSON(w, http.StatusForbidden, map[string]bool{"eligible": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"eligible": true})
}

// GET /api/v1/premium-check
func handlePremiumCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	apiKey := r.Header.Get("X-Api-Key")
	if apiKey != "test-secret-key" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"premium": true,
		"tier":    "gold",
	})
}

// POST /api/v1/score
func handleScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var body struct {
		UserID string `json:"userId"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	h := fnv.New32a()
	fmt.Fprint(h, body.UserID)
	score := int(h.Sum32() % 100)

	writeJSON(w, http.StatusOK, map[string]any{
		"score":    score,
		"eligible": score > 50,
	})
}

// GET /api/v1/slow
func handleSlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	time.Sleep(5 * time.Second)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"delayed": true,
	})
}

// GET /api/v1/flaky
func handleFlaky(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	count := flakyCounter.Add(1)
	if count%2 == 0 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}
