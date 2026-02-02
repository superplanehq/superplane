package gitlab

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__NewClient(t *testing.T) {
	mockClient := &contexts.HTTPContext{}

	t.Run("valid configuration - personal access token", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":            AuthTypePersonalAccessToken,
				"baseUrl":             "https://gitlab.example.com",
				"personalAccessToken": "pat-123",
				"groupId":             "group-123",
			},
		}

		client, err := NewClient(mockClient, ctx)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://gitlab.example.com", client.baseURL)
		assert.Equal(t, "pat-123", client.token)
		assert.Equal(t, AuthTypePersonalAccessToken, client.authType)
		assert.Equal(t, "group-123", client.groupID)
		assert.Equal(t, mockClient, client.httpClient)
	})

	t.Run("valid configuration - oauth", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAppOAuth,
				"groupId":  "group-456",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("oauth-token-123")},
			},
		}

		client, err := NewClient(mockClient, ctx)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://gitlab.com", client.baseURL) // Default
		assert.Equal(t, "oauth-token-123", client.token)
		assert.Equal(t, AuthTypeAppOAuth, client.authType)
		assert.Equal(t, "group-456", client.groupID)
	})

	t.Run("missing authType", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get authType")
	})

	t.Run("missing groupId", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypePersonalAccessToken,
			},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "groupId is required")
	})

	t.Run("missing personal access token", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypePersonalAccessToken,
				"groupId":  "123",
			},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
	})
}

func Test__Client__Verify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 123}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		err := client.Verify()
		require.NoError(t, err)
		
		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))
	})

	t.Run("forbidden", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusForbidden, `{"error": "fraud"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		err := client.Verify()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 403")
	})

	t.Run("oauth headers", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 123}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "oauth-token",
			authType:   AuthTypeAppOAuth,
			groupID:    "123",
			httpClient: mockClient,
		}

		err := client.Verify()
		require.NoError(t, err)
		
		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, "Bearer oauth-token", mockClient.Requests[0].Header.Get("Authorization"))
	})
}
