package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPublicHost = "mcp.example.com"

func TestDiscoverUsesWWWAuthenticate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="mcp", resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource"`)
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resource":              "https://mcp.example.com/mcp",
			"authorization_servers": []string{"https://mcp.example.com"},
		})
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 "https://mcp.example.com",
			"authorization_endpoint": "https://mcp.example.com/authorize",
			"token_endpoint":         "https://mcp.example.com/token",
			"registration_endpoint":  "https://mcp.example.com/register",
		})
	})

	client := rewriteClient(t, mux)
	discovery, err := Discover(context.Background(), client, "https://mcp.example.com/mcp")
	require.NoError(t, err)
	assert.Equal(t, "https://mcp.example.com/authorize", discovery.AuthServer.AuthorizationEndpoint)
	assert.Equal(t, "https://mcp.example.com/token", discovery.AuthServer.TokenEndpoint)
}

func TestSelectClientIDPrefersCIMDThenDCR(t *testing.T) {
	t.Parallel()
	cimd := &Discovery{AuthServer: AuthorizationServerMetadata{ClientIDMetadataDocumentSupported: true}}
	id, useDCR, err := cimd.SelectClientID("https://app.example/.well-known/oauth-client")
	require.NoError(t, err)
	assert.False(t, useDCR)
	assert.Equal(t, "https://app.example/.well-known/oauth-client", id)

	dcr := &Discovery{AuthServer: AuthorizationServerMetadata{RegistrationEndpoint: "https://auth.example/register"}}
	id, useDCR, err = dcr.SelectClientID("https://app.example/.well-known/oauth-client")
	require.NoError(t, err)
	assert.True(t, useDCR)
	assert.Empty(t, id)

	_, _, err = (&Discovery{}).SelectClientID("https://app.example/.well-known/oauth-client")
	require.Error(t, err)
}

func TestAuthorizationURLIncludesPKCEAndResource(t *testing.T) {
	t.Parallel()
	pkce := PKCE{Challenge: "abc", ChallengeMethod: "S256"}
	raw, err := AuthorizationURL("https://auth.example/authorize", "client", "https://app.example/callback", "state-1", "https://api.mobbin.com/mcp", pkce)
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	query := parsed.Query()
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, "abc", query.Get("code_challenge"))
	assert.Equal(t, "https://api.mobbin.com/mcp", query.Get("resource"))
}

func TestExchangeCodePostsVerifier(t *testing.T) {
	var posted url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		posted, _ = url.ParseQuery(string(body))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"expires_in":    3600,
		})
	})
	client := rewriteClient(t, mux)

	tokens, err := ExchangeCode(context.Background(), client, "https://mcp.example.com/token", "client", "", "https://app.example/callback", "code-1", "verifier", "https://api.mobbin.com/mcp")
	require.NoError(t, err)
	assert.Equal(t, "access", tokens.AccessToken)
	assert.Equal(t, "code-1", posted.Get("code"))
	assert.Equal(t, "verifier", posted.Get("code_verifier"))
	assert.Equal(t, "https://api.mobbin.com/mcp", posted.Get("resource"))
}

func TestResourceMetadataFromWWWAuthenticate(t *testing.T) {
	t.Parallel()
	got := resourceMetadataFromWWWAuthenticate(`Bearer FAKESECRET_g3h4i5j6k7l8m9n0o1p2="https://api.mobbin.com/.well-known/oauth-protected-resource"`)
	assert.Equal(t, "https://api.mobbin.com/.well-known/oauth-protected-resource", got)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func rewriteClient(t *testing.T, handler http.Handler) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			cloned := req.Clone(req.Context())
			if cloned.URL.Hostname() == testPublicHost {
				cloned.URL.Scheme = target.Scheme
				cloned.URL.Host = target.Host
			}
			return http.DefaultTransport.RoundTrip(cloned)
		}),
	}
}
