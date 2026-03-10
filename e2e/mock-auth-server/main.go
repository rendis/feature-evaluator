package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const (
	issuer = "http://localhost:9998/auth"
	kid    = "mock-key-1"
	port   = 9998
)

var privateKey *rsa.PrivateKey

func main() {
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key: %v", err)
	}
	log.Printf("RSA 2048-bit keypair generated (kid=%s)", kid)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/.well-known/openid-configuration", handleDiscovery)
	mux.HandleFunc("GET /auth/certs", handleJWKS)
	mux.HandleFunc("POST /auth/token", handleToken)
	mux.HandleFunc("POST /auth/validate", handleValidate)
	mux.HandleFunc("POST /auth/introspect", handleIntrospect)
	mux.HandleFunc("GET /auth/health", handleHealth)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("mock auth server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// --- OIDC Discovery ---

func handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	doc := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"jwks_uri":                              issuer + "/certs",
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"response_types_supported":              []string{"code"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
	}
	writeJSON(w, http.StatusOK, doc)
}

// --- JWKS ---

func handleJWKS(w http.ResponseWriter, _ *http.Request) {
	pub := &privateKey.PublicKey
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": kid,
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
	writeJSON(w, http.StatusOK, jwks)
}

// --- Token ---

type tokenRequest struct {
	Subject string `json:"subject"`
	Email   string `json:"email"`
	Role    string `json:"role"`
}

func handleToken(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	if req.Subject == "" {
		req.Subject = "mock-user"
	}
	if req.Email == "" {
		req.Email = "mock@test.com"
	}
	if req.Role == "" {
		req.Role = "admin"
	}

	name := req.Email
	if idx := strings.Index(req.Email, "@"); idx > 0 {
		name = req.Email[:idx]
	}

	now := time.Now()
	claims := map[string]any{
		"iss":   issuer,
		"sub":   req.Subject,
		"email": req.Email,
		"name":  name,
		"aud":   []string{"feature-evaluator"},
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
	}

	token, err := signJWT(claims)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "signing_failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// --- Validate ---

func handleValidate(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if suffix, found := strings.CutPrefix(token, "valid-token-"); found && suffix != "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid": true,
			"user": map[string]string{
				"id":    "user-" + suffix,
				"email": suffix + "@test.com",
				"role":  extractRole(suffix),
			},
		})
		return
	}

	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"valid": false,
		"error": "invalid_token",
	})
}

// extractToken resolves the bearer token from Authorization header or X-Token header.
// The custom_auth profile strips "Bearer " and forwards the raw token as X-Token.
func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		if token, ok := strings.CutPrefix(auth, "Bearer "); ok && strings.TrimSpace(token) != "" {
			return strings.TrimSpace(token)
		}
	}
	if xToken := strings.TrimSpace(r.Header.Get("X-Token")); xToken != "" {
		return xToken
	}
	return ""
}

// --- Introspect ---

func handleIntrospect(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(auth, "Bearer "); ok {
		if suffix, found := strings.CutPrefix(token, "valid-token-"); found && suffix != "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"active": true,
				"sub":    "user-" + suffix,
				"email":  suffix + "@test.com",
				"role":   extractRole(suffix),
			})
			return
		}
	}

	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"active": false,
		"error":  "invalid_token",
	})
}

// --- Health ---

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// --- JWT signing (standard library only) ---

func signJWT(claims map[string]any) (string, error) {
	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}

// --- Helpers ---

func extractRole(suffix string) string {
	switch suffix {
	case "admin":
		return "admin"
	case "user":
		return "user"
	case "viewer":
		return "viewer"
	default:
		return suffix
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
