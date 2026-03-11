package incomingauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
)

func TestValidateDraftOIDCStandardAcceptsValidJWT(t *testing.T) {
	t.Parallel()

	serverURL, privateKey := newOIDCTestServer(t)
	validator := NewValidator(nil, nil)

	token := mustSignJWT(t, privateKey, "test-kid", serverURL, "feature-evaluator", time.Now().Add(time.Hour))
	result, err := validator.ValidateDraft(context.Background(), &authprofile.Profile{
		Type: authprofile.TypeOIDCStandard,
		Config: map[string]any{
			"issuer":   serverURL,
			"audience": "feature-evaluator",
		},
	}, nil, map[string]any{
		"headers": map[string]any{
			"authorization": "Bearer " + token,
		},
	})
	if err != nil {
		t.Fatalf("ValidateDraft() error = %v", err)
	}
	if !result.Authenticated || !result.Attempted {
		t.Fatalf("ValidateDraft() = %+v, want authenticated attempted result", result.AuthValidationResult)
	}
}

func TestValidateDraftOIDCStandardRejectsAudienceMismatch(t *testing.T) {
	t.Parallel()

	serverURL, privateKey := newOIDCTestServer(t)
	validator := NewValidator(nil, nil)

	token := mustSignJWT(t, privateKey, "test-kid", serverURL, "wrong-audience", time.Now().Add(time.Hour))
	result, err := validator.ValidateDraft(context.Background(), &authprofile.Profile{
		Type: authprofile.TypeOIDCStandard,
		Config: map[string]any{
			"issuer":   serverURL,
			"audience": "feature-evaluator",
		},
	}, nil, map[string]any{
		"headers": map[string]any{
			"authorization": "Bearer " + token,
		},
	})
	if err != nil {
		t.Fatalf("ValidateDraft() error = %v", err)
	}
	if result.Authenticated {
		t.Fatalf("Authenticated = true, want false")
	}
	if !result.Attempted {
		t.Fatalf("Attempted = false, want true")
	}
	if got := fmt.Sprint(result.Details["reason"]); got == "" {
		t.Fatal("expected rejection reason in details")
	}
}

func newOIDCTestServer(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	var serverURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"jwks_uri": serverURL + "/keys",
			})
		case "/keys":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]any{
					{
						"kty": "RSA",
						"use": "sig",
						"kid": "test-kid",
						"alg": "RS256",
						"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
						"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL = server.URL
	return server.URL, privateKey
}

func mustSignJWT(t *testing.T, key *rsa.PrivateKey, kid, issuer, audience string, exp time.Time) string {
	t.Helper()

	header, err := json.Marshal(map[string]any{
		"alg": "RS256",
		"kid": kid,
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("json.Marshal(header) error = %v", err)
	}

	payload, err := json.Marshal(map[string]any{
		"iss":   issuer,
		"aud":   audience,
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "OIDC User",
		"exp":   exp.Unix(),
		"iat":   time.Now().Add(-time.Minute).Unix(),
		"nbf":   time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("json.Marshal(payload) error = %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signedContent := encodedHeader + "." + encodedPayload

	digest := sha256.Sum256([]byte(signedContent))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}

	return signedContent + "." + base64.RawURLEncoding.EncodeToString(signature)
}
