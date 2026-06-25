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
)

func TestPublicKeysFromJWKsRoundTrip(t *testing.T) {
	t.Parallel()

	provider, err := NewProviderFromKeyDir("http://superplane.test", filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	publicKeys, err := PublicKeysFromJWKs(provider.PublicJWKs())
	require.NoError(t, err)

	token, err := provider.Sign("execution:test", time.Hour, "superplane-ci", map[string]any{
		"org_id": uuid.NewString(),
	})
	require.NoError(t, err)

	claims, err := ValidateToken(token, "http://superplane.test", publicKeys)
	require.NoError(t, err)
	require.NotEmpty(t, claims["org_id"])
}

func TestValidateRemote(t *testing.T) {
	t.Parallel()

	issuer := "http://superplane.test"
	provider, err := NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			respondJSON(w, discoveryDocument{
				Issuer:  issuer,
				JWKSURI: server.URL + "/.well-known/jwks.json",
			})
		case "/.well-known/jwks.json":
			respondJSON(w, jwksDocument{Keys: provider.PublicJWKs()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	canvasID := uuid.NewString()
	token, err := provider.Sign(
		fmt.Sprintf("execution:%s", uuid.NewString()),
		time.Hour,
		"superplane-ci",
		map[string]any{
			"canvas_id":     canvasID,
			"pipeline_file": ".semaphore/deploy.yml",
		},
	)
	require.NoError(t, err)

	claims, err := ValidateRemote(server.Client(), token, server.URL)
	require.NoError(t, err)
	require.Equal(t, canvasID, claims["canvas_id"])
	require.Equal(t, ".semaphore/deploy.yml", claims["pipeline_file"])
}

func TestValidateRemoteRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	issuer := "http://superplane.test"
	provider, err := NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "test", "fixtures", "oidc-keys"))
	require.NoError(t, err)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			respondJSON(w, discoveryDocument{Issuer: issuer, JWKSURI: server.URL + "/.well-known/jwks.json"})
		case "/.well-known/jwks.json":
			respondJSON(w, jwksDocument{Keys: provider.PublicJWKs()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err = ValidateRemote(server.Client(), "not-a-token", server.URL)
	require.Error(t, err)
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
