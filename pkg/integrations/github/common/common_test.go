package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__EnsureRepoInMetadata__NewSetupFlow(t *testing.T) {
	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			httpCtx,
			map[string]any{"repository": ""},
		)

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "world"},
		)

		require.ErrorContains(t, err, "failed to find repository")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/world", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "Bearer test-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("repository exists and is saved in metadata", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		err := EnsureRepoInMetadata(
			nodeMetadataCtx,
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "hello"},
		)

		require.NoError(t, err)

		helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "Bearer test-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})
}

func Test__EnsureRepoInMetadata__LegacySetupFlow(t *testing.T) {
	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForLegacySetupFlow(testPrivateKeyPEM(t))

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			&contexts.HTTPContext{},
			map[string]any{"repository": ""},
		)

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForLegacySetupFlow(testPrivateKeyPEM(t))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(
					http.StatusCreated,
					fmt.Sprintf(`{"token":"test-installation-token","expires_at":%q}`, time.Now().Add(time.Hour).Format(time.RFC3339)),
				),
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "world"},
		)

		require.ErrorContains(t, err, "failed to find repository")
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/app/installations/67890/access_tokens", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Equal(t, "/repos/testhq/world", httpCtx.Requests[1].URL.Path)
		assert.Equal(t, "token test-installation-token", httpCtx.Requests[1].Header.Get("Authorization"))
	})

	t.Run("repository exists and is saved in metadata", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForLegacySetupFlow(testPrivateKeyPEM(t))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(
					http.StatusCreated,
					fmt.Sprintf(`{"token":"test-installation-token","expires_at":%q}`, time.Now().Add(time.Hour).Format(time.RFC3339)),
				),
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		err := EnsureRepoInMetadata(
			nodeMetadataCtx,
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "hello"},
		)

		require.NoError(t, err)

		helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/app/installations/67890/access_tokens", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
		assert.Equal(t, "/repos/testhq/hello", httpCtx.Requests[1].URL.Path)
		assert.Equal(t, "token test-installation-token", httpCtx.Requests[1].Header.Get("Authorization"))
	})
}

func testPrivateKeyPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}
