package middleware

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// JWTValidator validates JWT tokens against OIDC JWKS.
type JWTValidator struct {
	issuer   string
	audience string

	mu        sync.RWMutex
	jwks      map[string]crypto.PublicKey
	lastFetch time.Time
	ttl       time.Duration
	client    *http.Client
}

// NewJWTValidator creates a new JWTValidator.
func NewJWTValidator(issuer, audience string) *JWTValidator {
	return &JWTValidator{
		issuer:   strings.TrimRight(issuer, "/"),
		audience: audience,
		jwks:     make(map[string]crypto.PublicKey),
		ttl:      10 * time.Minute,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// JWTClaims holds the validated claims extracted from a JWT.
type JWTClaims struct {
	Subject string
	Email   string
	Name    string
}

// Validate validates a JWT token and returns extracted claims.
func (v *JWTValidator) Validate(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed JWT: expected 3 parts")
	}

	headerBytes, err := base64URLDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parsing JWT header: %w", err)
	}

	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}
	var claims struct {
		Iss   string `json:"iss"`
		Aud   any    `json:"aud"`
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Exp   int64  `json:"exp"`
		Iat   int64  `json:"iat"`
		Nbf   int64  `json:"nbf"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	// Validate issuer
	claimIss := strings.TrimRight(claims.Iss, "/")
	if claimIss != v.issuer {
		return nil, fmt.Errorf("invalid issuer: got %q, want %q", claims.Iss, v.issuer)
	}

	// Validate audience
	if !v.matchAudience(claims.Aud) {
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate expiration
	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	if claims.Nbf > 0 && now < claims.Nbf {
		return nil, fmt.Errorf("token not yet valid")
	}

	// Verify signature
	signatureBytes, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT signature: %w", err)
	}
	signedContent := parts[0] + "." + parts[1]

	key, err := v.getKey(header.Kid)
	if err != nil {
		return nil, fmt.Errorf("getting signing key: %w", err)
	}

	if err := verifySignature(header.Alg, key, []byte(signedContent), signatureBytes); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &JWTClaims{
		Subject: claims.Sub,
		Email:   claims.Email,
		Name:    claims.Name,
	}, nil
}

func (v *JWTValidator) matchAudience(aud any) bool {
	if v.audience == "" {
		return true
	}
	switch a := aud.(type) {
	case string:
		return a == v.audience
	case []any:
		for _, item := range a {
			if s, ok := item.(string); ok && s == v.audience {
				return true
			}
		}
	}
	return false
}

func (v *JWTValidator) getKey(kid string) (crypto.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.jwks[kid]; ok && time.Since(v.lastFetch) < v.ttl {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	if err := v.fetchJWKS(); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.jwks[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

//nolint:cyclop,funlen // JWKS discovery handles provider config, cache refresh, and multiple key types in one flow.
func (v *JWTValidator) fetchJWKS() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(v.lastFetch) < v.ttl && len(v.jwks) > 0 {
		return nil
	}

	// Discover JWKS URI from OIDC config
	configURL := v.issuer + "/.well-known/openid-configuration"
	resp, err := v.client.Get(configURL)
	if err != nil {
		return fmt.Errorf("fetching OIDC config: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading OIDC config: %w", err)
	}

	var config struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("parsing OIDC config: %w", err)
	}

	if config.JWKSURI == "" {
		return errors.New("OIDC config missing jwks_uri")
	}

	// Fetch JWKS
	jwksResp, err := v.client.Get(config.JWKSURI)
	if err != nil {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	defer jwksResp.Body.Close()
	jwksBody, err := io.ReadAll(io.LimitReader(jwksResp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("reading JWKS: %w", err)
	}

	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(jwksBody, &jwks); err != nil {
		return fmt.Errorf("parsing JWKS: %w", err)
	}

	newKeys := make(map[string]crypto.PublicKey, len(jwks.Keys))
	for _, raw := range jwks.Keys {
		var key struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			// RSA
			N string `json:"n"`
			E string `json:"e"`
			// EC
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
			// x509
			X5c []string `json:"x5c"`
		}
		if err := json.Unmarshal(raw, &key); err != nil {
			continue
		}

		var pubKey crypto.PublicKey
		switch key.Kty {
		case "RSA":
			pubKey = parseRSAKey(key.N, key.E)
		case "EC":
			pubKey = parseECKey(key.Crv, key.X, key.Y)
		default:
			if len(key.X5c) > 0 {
				pubKey = parseX5C(key.X5c[0])
			}
		}

		if pubKey != nil && key.Kid != "" {
			newKeys[key.Kid] = pubKey
		}
	}

	v.jwks = newKeys
	v.lastFetch = time.Now()
	return nil
}

func parseRSAKey(nStr, eStr string) *rsa.PublicKey {
	nBytes, err := base64URLDecode(nStr)
	if err != nil {
		return nil
	}
	eBytes, err := base64URLDecode(eStr)
	if err != nil {
		return nil
	}
	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	return &rsa.PublicKey{N: n, E: e}
}

func parseECKey(crv, xStr, yStr string) *ecdsa.PublicKey {
	xBytes, err := base64URLDecode(xStr)
	if err != nil {
		return nil
	}
	yBytes, err := base64URLDecode(yStr)
	if err != nil {
		return nil
	}
	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil
	}
	return &ecdsa.PublicKey{Curve: curve, X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}
}

func parseX5C(certStr string) crypto.PublicKey {
	certBytes, err := base64.StdEncoding.DecodeString(certStr)
	if err != nil {
		block, _ := pem.Decode([]byte("-----BEGIN CERTIFICATE-----\n" + certStr + "\n-----END CERTIFICATE-----"))
		if block == nil {
			return nil
		}
		certBytes = block.Bytes
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil
	}
	return cert.PublicKey
}

func verifySignature(alg string, key crypto.PublicKey, signed, signature []byte) error {
	switch alg {
	case "RS256":
		return verifyRSA(crypto.SHA256, key, signed, signature)
	case "RS384":
		return verifyRSA(crypto.SHA384, key, signed, signature)
	case "RS512":
		return verifyRSA(crypto.SHA512, key, signed, signature)
	case "ES256":
		return verifyECDSA(crypto.SHA256, key, signed, signature)
	case "ES384":
		return verifyECDSA(crypto.SHA384, key, signed, signature)
	case "ES512":
		return verifyECDSA(crypto.SHA512, key, signed, signature)
	default:
		return fmt.Errorf("unsupported algorithm: %s", alg)
	}
}

func verifyRSA(hash crypto.Hash, key crypto.PublicKey, signed, signature []byte) error {
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return errors.New("key is not RSA")
	}
	h := hash.New()
	h.Write(signed)
	return rsa.VerifyPKCS1v15(rsaKey, hash, h.Sum(nil), signature)
}

func verifyECDSA(hash crypto.Hash, key crypto.PublicKey, signed, signature []byte) error {
	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("key is not ECDSA")
	}
	h := hash.New()
	h.Write(signed)
	if !ecdsa.VerifyASN1(ecKey, h.Sum(nil), signature) {
		return errors.New("ECDSA signature verification failed")
	}
	return nil
}

func base64URLDecode(s string) ([]byte, error) {
	// Pad if necessary
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
