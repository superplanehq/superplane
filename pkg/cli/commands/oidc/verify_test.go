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

func TestValidateRemote(t *testing.T) {
	t.Parallel()

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
	defer server.Close()

	nodeID := uuid.NewString()
	token := signTestExecutionToken(t, server.URL, map[string]any{
		"node_id":       nodeID,
		"pipeline_file": ".semaphore/deploy.yml",
	})

	claims, err := validateRemote(t.Context(), server.Client(), token, server.URL)
	require.NoError(t, err)
	require.Equal(t, nodeID, claims["node_id"])
	require.Equal(t, ".semaphore/deploy.yml", claims["pipeline_file"])
}

func TestValidateRemoteRejectsInvalidToken(t *testing.T) {
	t.Parallel()

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
	defer server.Close()

	_, err := validateRemote(t.Context(), server.Client(), "not-a-token", server.URL)
	require.Error(t, err)
}

func TestParseExpectedClaims(t *testing.T) {
	t.Parallel()

	expected, err := parseExpectedClaims([]string{
		"pipeline_file=.semaphore/deploy.yml",
		"node_id=deploy",
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, expected)

	_, err = parseExpectedClaims([]string{"invalid"})
	require.Error(t, err)
}

func TestMatchExpectedClaims(t *testing.T) {
	t.Parallel()

	err := matchExpectedClaims(map[string]any{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	}, map[string]string{
		"pipeline_file": ".semaphore/deploy.yml",
		"node_id":       "deploy",
	})
	require.NoError(t, err)

	err = matchExpectedClaims(map[string]any{
		"node_id": "other-node",
	}, map[string]string{
		"node_id": "expected-node",
	})
	require.Error(t, err)
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
		executionTokenAudience,
		claims,
	)
	require.NoError(t, err)
	return token
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
