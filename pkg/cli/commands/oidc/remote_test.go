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

	issuer := "http://superplane.test"
	provider, err := spoidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "..", "..", "test", "fixtures", "oidc-keys"))
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

	claims, err := validateRemote(server.Client(), token, server.URL)
	require.NoError(t, err)
	require.Equal(t, canvasID, claims["canvas_id"])
	require.Equal(t, ".semaphore/deploy.yml", claims["pipeline_file"])
}

func TestValidateRemoteRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	issuer := "http://superplane.test"
	provider, err := spoidc.NewProviderFromKeyDir(issuer, filepath.Join("..", "..", "..", "..", "test", "fixtures", "oidc-keys"))
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

	_, err = validateRemote(server.Client(), "not-a-token", server.URL)
	require.Error(t, err)
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
