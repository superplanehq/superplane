package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	spoidc "github.com/superplanehq/superplane/pkg/oidc"
)

const testExecutionTokenAudience = "semaphore"

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	nodeID := uuid.NewString()
	server, token := newOIDCTestServer(t, map[string]any{
		"node_id":       nodeID,
		"pipeline_file": ".semaphore/deploy.yml",
	})

	cmd := &verifyCommand{
		client:         server.Client(),
		parsedToken:    token,
		parsedAPIURL:   server.URL,
		parsedAudience: testExecutionTokenAudience,
		parsedExpectedClaims: map[string]string{
			"node_id":       nodeID,
			"pipeline_file": ".semaphore/deploy.yml",
		},
	}

	require.NoError(t, cmd.verifyToken(t.Context()))
}

func TestVerifyTokenRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	server, _ := newOIDCTestServer(t, nil)

	cmd := &verifyCommand{
		client:               server.Client(),
		parsedToken:          "not-a-token",
		parsedAPIURL:         server.URL,
		parsedAudience:       testExecutionTokenAudience,
		parsedExpectedClaims: map[string]string{},
	}

	require.Error(t, cmd.verifyToken(t.Context()))
}

func TestVerifyTokenRejectsMismatchedClaim(t *testing.T) {
	t.Parallel()

	nodeID := uuid.NewString()
	server, token := newOIDCTestServer(t, map[string]any{
		"node_id": nodeID,
	})

	cmd := &verifyCommand{
		client:         server.Client(),
		parsedToken:    token,
		parsedAPIURL:   server.URL,
		parsedAudience: testExecutionTokenAudience,
		parsedExpectedClaims: map[string]string{
			"node_id": "other-node",
		},
	}

	err := cmd.verifyToken(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected claim node_id")
}

func TestParseExpectedClaims(t *testing.T) {
	t.Parallel()

	raw := []string{
		"pipeline_file=.semaphore/deploy.yml",
		"node_id=deploy",
	}
	cmd := &verifyCommand{expectedClaims: &raw}

	require.NoError(t, cmd.parseExpectedClaims())
	require.Equal(t, map[string]string{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, cmd.parsedExpectedClaims)

	invalid := []string{"invalid"}
	cmd = &verifyCommand{expectedClaims: &invalid}
	require.Error(t, cmd.parseExpectedClaims())
}

func TestClaimString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "deploy", claimString(map[string]any{"node_id": "deploy"}, "node_id"))
	require.Equal(t, "", claimString(map[string]any{}, "node_id"))
	require.Equal(t, "123", claimString(map[string]any{"count": 123}, "count"))
}

func newOIDCTestServer(t *testing.T, claims map[string]any) (*httptest.Server, string) {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			respondJSON(w, map[string]any{
				"issuer":   server.URL,
				"jwks_uri": server.URL + "/.well-known/jwks.json",
			})
		case "/.well-known/jwks.json":
			respondJSON(w, map[string]any{"keys": providerPublicJWKs(t, server.URL)})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	token := signTestExecutionToken(t, server.URL, claims)
	return server, token
}

func providerPublicJWKs(t *testing.T, issuer string) []spoidc.PublicJWK {
	t.Helper()

	provider, err := spoidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)
	return provider.PublicJWKs()
}

func signTestExecutionToken(t *testing.T, issuer string, claims map[string]any) string {
	t.Helper()

	provider, err := spoidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	token, err := provider.Sign(
		fmt.Sprintf("execution:%s", uuid.NewString()),
		time.Hour,
		testExecutionTokenAudience,
		claims,
	)
	require.NoError(t, err)
	return token
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
