package mcp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestMintReusesAccessTokenUntilNearExpiry(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(time.Hour), "cached-access", "refresh-1")
	calls := 0
	client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
		calls++
		http.Error(w, "refresh was not required", http.StatusInternalServerError)
	})

	token, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
	require.NoError(t, err)
	assert.Equal(t, "cached-access", token)
	assert.Zero(t, calls)
	assert.Equal(t, models.FactoryAgentResourceOAuthConnected, reloadedOAuthStatus(t, db, resource))
}

func TestMintRefreshesOnceWhenAccessTokenIsNearExpiry(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(30*time.Second), "old-access", "refresh-1")
	var calls int
	var callsMu sync.Mutex
	client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
		callsMu.Lock()
		calls++
		callsMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"refresh-2","expires_in":3600,"token_type":"Bearer"}`)
	})

	token, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
	require.NoError(t, err)
	assert.Equal(t, "new-access", token)

	token, err = mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
	require.NoError(t, err)
	assert.Equal(t, "new-access", token)
	callsMu.Lock()
	assert.Equal(t, 1, calls)
	callsMu.Unlock()

	saved := reloadResource(t, db, resource)
	assert.Equal(t, models.FactoryAgentResourceOAuthConnected, saved.OAuthState())
	expiresAt, err := time.Parse(time.RFC3339, saved.OAuthMetadata.Data().AccessTokenExpiresAt)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, 2*time.Minute)
	assert.Equal(t, "refresh-2", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretRefreshToken))
	assert.Equal(t, "new-access", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretAccessToken))
}

func TestMintKeepsConnectionWhenRefreshFailsTemporarily(t *testing.T) {
	t.Run("expired token returns the error", func(t *testing.T) {
		r := support.Setup(t)
		db := database.DB(t.Context())
		resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "old-access", "refresh-1")
		client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "unavailable", http.StatusInternalServerError)
		})

		_, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
		require.Error(t, err)
		saved := reloadResource(t, db, resource)
		assert.Equal(t, models.FactoryAgentResourceOAuthConnected, saved.OAuthState())
		assert.Empty(t, saved.OAuthError)
	})

	t.Run("token inside the refresh window is reused", func(t *testing.T) {
		r := support.Setup(t)
		db := database.DB(t.Context())
		resource := connectOAuthResource(t, db, r, time.Now().Add(30*time.Second), "still-good", "refresh-1")
		client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "unavailable", http.StatusInternalServerError)
		})

		token, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
		require.NoError(t, err)
		assert.Equal(t, "still-good", token)
		assert.Equal(t, models.FactoryAgentResourceOAuthConnected, reloadedOAuthStatus(t, db, resource))
	})
}

func TestMintMarksReconnectWhenGrantIsRejected(t *testing.T) {
	for _, code := range []string{"invalid_grant", "invalid_client"} {
		t.Run(code, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "old-access", "refresh-1")
			client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"`+code+`","error_description":"Invalid refresh token"}`)
			})

			_, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
			require.Error(t, err)
			saved := reloadResource(t, db, resource)
			assert.Equal(t, models.FactoryAgentResourceOAuthNeedsReconnect, saved.OAuthState())
			assert.Contains(t, saved.OAuthError, code)
			assert.Equal(t, "refresh-1", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretRefreshToken))
		})
	}
}

func TestMintAdoptsRefreshTokenSavedByAnotherCaller(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "old-access", "refresh-1")
	var handlerErr error
	client := tokenEndpointClient(t, func(w http.ResponseWriter, req *http.Request) {
		if err := storePlainSecret(context.Background(), r, database.Conn(), resource, models.FactoryAgentResourceSecretRefreshToken, "refresh-2"); err != nil {
			handlerErr = err
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := storePlainSecret(context.Background(), r, database.Conn(), resource, models.FactoryAgentResourceSecretAccessToken, "winner-access"); err != nil {
			handlerErr = err
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"Invalid refresh token"}`)
	})

	token, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, resource)
	require.NoError(t, handlerErr)
	require.NoError(t, err)
	assert.Equal(t, "winner-access", token)
	saved := reloadResource(t, db, resource)
	assert.Equal(t, models.FactoryAgentResourceOAuthConnected, saved.OAuthState())
	assert.Empty(t, saved.OAuthError)
}

func connectOAuthResource(t *testing.T, db *gorm.DB, r *support.ResourceRegistry, expiresAt time.Time, accessToken, refreshToken string) *models.FactoryAgentResource {
	t.Helper()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "sentry", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		TokenEndpoint:        "https://mcp.example.com/token",
		ClientID:             "client-1",
		AccessTokenExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	}))
	require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	require.NoError(t, storePlainSecret(t.Context(), r, db, resource, models.FactoryAgentResourceSecretAccessToken, accessToken))
	require.NoError(t, storePlainSecret(t.Context(), r, db, resource, models.FactoryAgentResourceSecretRefreshToken, refreshToken))
	return resource
}

func storePlainSecret(ctx context.Context, r *support.ResourceRegistry, db *gorm.DB, resource *models.FactoryAgentResource, name, value string) error {
	encrypted, err := mcp.EncryptResourceSecret(ctx, r.Encryptor, resource.ID, value)
	if err != nil {
		return err
	}
	return resource.UpsertSecret(db, name, encrypted)
}

func decryptedSecret(t *testing.T, r *support.ResourceRegistry, db *gorm.DB, resource *models.FactoryAgentResource, name string) string {
	t.Helper()
	value, err := mcp.DecryptedResourceSecret(t.Context(), r.Encryptor, db, resource, name)
	require.NoError(t, err)
	return value
}

func reloadResource(t *testing.T, db *gorm.DB, resource *models.FactoryAgentResource) *models.FactoryAgentResource {
	t.Helper()
	factory := &models.Factory{ID: resource.FactoryID, OrganizationID: resource.OrganizationID}
	saved, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	return saved
}

func reloadedOAuthStatus(t *testing.T, db *gorm.DB, resource *models.FactoryAgentResource) string {
	t.Helper()
	return reloadResource(t, db, resource).OAuthState()
}

func tokenEndpointClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	return &http.Client{
		Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
			cloned := req.Clone(req.Context())
			if cloned.URL.Hostname() == "mcp.example.com" {
				cloned.URL.Scheme = target.Scheme
				cloned.URL.Host = target.Host
			}
			return http.DefaultTransport.RoundTrip(cloned)
		}),
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
