package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ClientName       = "SuperPlane"
	OAuthTimeout     = 15 * time.Second
	MaxMetadataBytes = 1 << 20
	DefaultTokenTTL  = time.Hour
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
}

type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
}

type ClientRegistration struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type PKCE struct {
	Verifier        string
	Challenge       string
	ChallengeMethod string
}

type Discovery struct {
	Resource      string
	Protected     ProtectedResourceMetadata
	AuthServer    AuthorizationServerMetadata
	AuthServerURL string
}

func NewPKCE() (PKCE, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return PKCE{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	return PKCE{
		Verifier:        verifier,
		Challenge:       base64.RawURLEncoding.EncodeToString(sum[:]),
		ChallengeMethod: "S256",
	}, nil
}

func RandomState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func Discover(ctx context.Context, httpClient HTTPDoer, mcpURL string) (*Discovery, error) {
	parsed, err := parseHTTPSURL(mcpURL)
	if err != nil {
		return nil, err
	}
	canonical := canonicalResource(parsed)

	metadataURL, err := probeResourceMetadataURL(ctx, httpClient, canonical)
	if err != nil {
		return nil, err
	}
	var protected ProtectedResourceMetadata
	if err := getJSON(ctx, httpClient, metadataURL, &protected); err != nil {
		return nil, fmt.Errorf("load protected resource metadata: %w", err)
	}
	if strings.TrimSpace(protected.Resource) == "" {
		protected.Resource = canonical
	}
	if err := ValidatePublicHTTPSURL(protected.Resource); err != nil {
		return nil, fmt.Errorf("protected resource URL is not valid")
	}
	if len(protected.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("the MCP server did not list an authorization server")
	}
	authServerURL := strings.TrimSpace(protected.AuthorizationServers[0])
	if err := ValidatePublicHTTPSURL(authServerURL); err != nil {
		return nil, fmt.Errorf("authorization server URL is not valid")
	}

	authMeta, err := discoverAuthServer(ctx, httpClient, authServerURL)
	if err != nil {
		return nil, err
	}
	if err := ValidatePublicHTTPSURL(authMeta.AuthorizationEndpoint); err != nil {
		return nil, fmt.Errorf("authorization endpoint is not valid")
	}
	if err := ValidatePublicHTTPSURL(authMeta.TokenEndpoint); err != nil {
		return nil, fmt.Errorf("token endpoint is not valid")
	}

	return &Discovery{
		Resource:      protected.Resource,
		Protected:     protected,
		AuthServer:    *authMeta,
		AuthServerURL: authServerURL,
	}, nil
}

func RegisterClient(ctx context.Context, httpClient HTTPDoer, registrationEndpoint, redirectURI, clientMetadataURL string) (*ClientRegistration, error) {
	if err := ValidatePublicHTTPSURL(registrationEndpoint); err != nil {
		return nil, fmt.Errorf("registration endpoint is not valid")
	}
	body := map[string]any{
		"client_name":                ClientName,
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"application_type":           "web",
	}
	if clientMetadataURL != "" {
		body["client_uri"] = clientMetadataURL
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationEndpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	var registration ClientRegistration
	if err := doJSON(httpClient, req, &registration); err != nil {
		return nil, fmt.Errorf("dynamic client registration failed: %w", err)
	}
	if strings.TrimSpace(registration.ClientID) == "" {
		return nil, fmt.Errorf("dynamic client registration did not return a client id")
	}
	return &registration, nil
}

func AuthorizationURL(authEndpoint, clientID, redirectURI, state, resource string, pkce PKCE) (string, error) {
	parsed, err := url.Parse(authEndpoint)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("code_challenge", pkce.Challenge)
	query.Set("code_challenge_method", pkce.ChallengeMethod)
	if resource != "" {
		query.Set("resource", resource)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func ExchangeCode(ctx context.Context, httpClient HTTPDoer, tokenEndpoint, clientID, clientSecret, redirectURI, code, verifier, resource string) (*TokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("redirect_uri", redirectURI)
	values.Set("client_id", clientID)
	values.Set("code_verifier", verifier)
	if resource != "" {
		values.Set("resource", resource)
	}
	return postToken(ctx, httpClient, tokenEndpoint, clientID, clientSecret, values)
}

func RefreshAccessToken(ctx context.Context, httpClient HTTPDoer, tokenEndpoint, clientID, clientSecret, refreshToken, resource string) (*TokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", refreshToken)
	values.Set("client_id", clientID)
	if resource != "" {
		values.Set("resource", resource)
	}
	return postToken(ctx, httpClient, tokenEndpoint, clientID, clientSecret, values)
}

func RevokeToken(ctx context.Context, httpClient HTTPDoer, revocationEndpoint, clientID, token string) {
	if strings.TrimSpace(revocationEndpoint) == "" || strings.TrimSpace(token) == "" {
		return
	}
	values := url.Values{}
	values.Set("token", token)
	values.Set("client_id", clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revocationEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func ClientMetadataDocument(clientMetadataURL, redirectURI string) map[string]any {
	return map[string]any{
		"client_id":                  clientMetadataURL,
		"client_name":                ClientName,
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"application_type":           "web",
	}
}

func CallbackURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/api/v1/mcp-oauth/callback"
}

func ClientMetadataURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/.well-known/oauth-client"
}

func (d *Discovery) SelectClientID(clientMetadataURL string) (clientID string, useDCR bool, err error) {
	if d.AuthServer.ClientIDMetadataDocumentSupported && clientMetadataURL != "" {
		return clientMetadataURL, false, nil
	}
	if strings.TrimSpace(d.AuthServer.RegistrationEndpoint) != "" {
		return "", true, nil
	}
	return "", false, fmt.Errorf("this MCP server does not accept SuperPlane as an OAuth client")
}

func TokenTTL(tokens *TokenResponse) time.Duration {
	if tokens == nil || tokens.ExpiresIn <= 0 {
		return DefaultTokenTTL
	}
	return time.Duration(tokens.ExpiresIn) * time.Second
}

func DoerFromCore(httpCtx core.HTTPContext) HTTPDoer {
	if httpCtx == nil {
		return http.DefaultClient
	}
	return httpCtx
}

func probeResourceMetadataURL(ctx context.Context, httpClient HTTPDoer, mcpURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mcpURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("probe MCP URL: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if headerURL := resourceMetadataFromWWWAuthenticate(resp.Header.Get("WWW-Authenticate")); headerURL != "" {
		if err := ValidatePublicHTTPSURL(headerURL); err != nil {
			return "", fmt.Errorf("resource metadata URL is not valid")
		}
		return headerURL, nil
	}

	parsed, err := url.Parse(mcpURL)
	if err != nil {
		return "", err
	}
	parsed.Path = "/.well-known/oauth-protected-resource"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	fallback := parsed.String()
	if err := ValidatePublicHTTPSURL(fallback); err != nil {
		return "", err
	}
	return fallback, nil
}

func resourceMetadataFromWWWAuthenticate(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	const key = "resource_metadata="
	lower := strings.ToLower(header)
	idx := strings.Index(lower, key)
	if idx < 0 {
		return ""
	}
	rest := header[idx+len(key):]
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, `"`) {
		rest = rest[1:]
		end := strings.Index(rest, `"`)
		if end < 0 {
			return ""
		}
		return rest[:end]
	}
	end := strings.IndexAny(rest, " ,;")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func discoverAuthServer(ctx context.Context, httpClient HTTPDoer, issuer string) (*AuthorizationServerMetadata, error) {
	parsed, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}
	candidates := []string{
		issuer + "/.well-known/oauth-authorization-server",
		issuer + "/.well-known/openid-configuration",
	}
	if parsed.Path != "" && parsed.Path != "/" {
		host := *parsed
		path := strings.TrimSuffix(parsed.Path, "/")
		host.Path = "/.well-known/oauth-authorization-server" + path
		candidates = append([]string{host.String()}, candidates...)
	}

	var lastErr error
	for _, candidate := range candidates {
		if err := ValidatePublicHTTPSURL(candidate); err != nil {
			lastErr = err
			continue
		}
		var meta AuthorizationServerMetadata
		if err := getJSON(ctx, httpClient, candidate, &meta); err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(meta.AuthorizationEndpoint) == "" || strings.TrimSpace(meta.TokenEndpoint) == "" {
			lastErr = fmt.Errorf("authorization server metadata is incomplete")
			continue
		}
		return &meta, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("authorization server metadata not found")
	}
	return nil, lastErr
}

func postToken(ctx context.Context, httpClient HTTPDoer, tokenEndpoint, clientID, clientSecret string, values url.Values) (*TokenResponse, error) {
	if err := ValidatePublicHTTPSURL(tokenEndpoint); err != nil {
		return nil, fmt.Errorf("token endpoint is not valid")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if clientSecret != "" {
		req.SetBasicAuth(clientID, clientSecret)
	}
	var tokens TokenResponse
	if err := doJSON(httpClient, req, &tokens); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tokens.AccessToken) == "" {
		return nil, fmt.Errorf("token response did not include an access token")
	}
	return &tokens, nil
}

func getJSON(ctx context.Context, httpClient HTTPDoer, rawURL string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	return doJSON(httpClient, req, dest)
}

func doJSON(httpClient HTTPDoer, req *http.Request, dest any) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, MaxMetadataBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func canonicalResource(parsed *url.URL) string {
	clone := *parsed
	clone.Fragment = ""
	if clone.Path == "" {
		clone.Path = "/"
	}
	return strings.TrimRight(clone.String(), "/")
}

func TimeoutContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, OAuthTimeout)
}
