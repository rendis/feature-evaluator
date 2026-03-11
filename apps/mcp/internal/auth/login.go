package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	loginTimeout = 5 * time.Minute

	defaultOIDCIssuer   = ""
	defaultOIDCClientID = "feature-evaluator"
	defaultOIDCScopes   = "openid profile email offline_access"
)

// oidcDiscoveryResponse is the standard OpenID Connect discovery response.
type oidcDiscoveryResponse struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// RunLogin performs browser-based OIDC login using Authorization Code + PKCE.
func RunLogin() error {
	issuer := os.Getenv("FE_OIDC_ISSUER")
	if issuer == "" {
		issuer = defaultOIDCIssuer
	}
	clientID := os.Getenv("FE_OIDC_CLIENT_ID")
	if clientID == "" {
		clientID = defaultOIDCClientID
	}

	fmt.Printf("Fetching OIDC configuration from %s...\n", issuer)

	discovery, err := fetchOIDCDiscovery(issuer)
	if err != nil {
		return fmt.Errorf("OIDC discovery: %w", err)
	}

	authEndpoint := discovery.AuthorizationEndpoint
	tokenEndpoint := discovery.TokenEndpoint

	if authEndpoint == "" || tokenEndpoint == "" {
		return fmt.Errorf("incomplete OIDC discovery: authorization_endpoint=%q, token_endpoint=%q",
			authEndpoint, tokenEndpoint)
	}

	scopes := defaultOIDCScopes

	// PKCE
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return fmt.Errorf("generate PKCE: %w", err)
	}

	state, err := randomString(32)
	if err != nil {
		return fmt.Errorf("generate state: %w", err)
	}

	// Start localhost callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("authorization error: %s — %s", errMsg, desc)
			_, _ = fmt.Fprintf(w, "<html><body><h2>Login failed</h2><p>%s: %s</p><p>You can close this tab.</p></body></html>", errMsg, desc)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code in callback")
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		_, _ = fmt.Fprint(w, "<html><body><h2>Login successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()

	// Build authorization URL
	authURL := buildAuthURL(authEndpoint, clientID, redirectURI, scopes, state, challenge)

	fmt.Printf("\nOpening browser for login...\n")
	fmt.Printf("If the browser doesn't open, visit this URL manually:\n\n  %s\n\n", authURL)

	_ = openBrowser(authURL)

	// Wait for callback or timeout
	ctx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	defer cancel()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		_ = srv.Shutdown(ctx)
		return err
	case <-ctx.Done():
		_ = srv.Shutdown(ctx)
		return fmt.Errorf("login timed out after %v", loginTimeout)
	}

	_ = srv.Shutdown(ctx)

	// Exchange code for tokens
	fmt.Println("Exchanging authorization code for tokens...")

	tokens, err := exchangeCode(tokenEndpoint, clientID, code, redirectURI, verifier)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}

	stored := &StoredTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		ExpiresAt:     time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
		TokenEndpoint: tokenEndpoint,
		ClientID:      clientID,
	}

	filePath := TokenFilePath()
	if err := SaveTokens(filePath, stored); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	fmt.Printf("\nLogin successful! Tokens saved to %s\n", filePath)
	if tokens.RefreshToken != "" {
		fmt.Println("Refresh token obtained — tokens will auto-refresh.")
	} else {
		fmt.Println("Warning: no refresh token received. You may need to login again when the token expires.")
		fmt.Println("Tip: ensure your OIDC provider returns refresh tokens (scope: offline_access).")
	}

	return nil
}

func fetchOIDCDiscovery(issuer string) (*oidcDiscoveryResponse, error) {
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", discoveryURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned %d", resp.StatusCode)
	}
	var discovery oidcDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, err
	}
	return &discovery, nil
}

func generatePKCE() (verifier, challenge string, err error) {
	raw, err := randomBytes(32)
	if err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

func randomString(n int) (string, error) {
	b, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func buildAuthURL(endpoint, clientID, redirectURI, scopes, state, challenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return endpoint + "?" + params.Encode()
}

func exchangeCode(tokenEndpoint, clientID, code, redirectURI, verifier string) (*tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.PostForm(tokenEndpoint, form)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token in response")
	}
	return &tok, nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
