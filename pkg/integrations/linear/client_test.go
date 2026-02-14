package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing access token -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{},
		}
		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access token")
	})

	t.Run("successful client creation", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("test-key")},
			},
		}
		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, "test-key", client.accessToken)
	})
}

func Test__Client__GetViewer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		resp := `{"data":{"viewer":{"id":"u1","name":"Alice","email":"a@b.com"}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusOK, resp),
			},
		}
		appCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("key")},
			},
		}
		client, err := NewClient(httpCtx, appCtx)
		require.NoError(t, err)
		viewer, err := client.GetViewer()
		require.NoError(t, err)
		assert.Equal(t, "u1", viewer.ID)
		assert.Equal(t, "Alice", viewer.Name)
	})
}

func Test__Client__ListTeams(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		resp := `{"data":{"teams":{"nodes":[{"id":"t1","name":"Eng","key":"ENG"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				linearMockResponse(http.StatusOK, resp),
			},
		}
		appCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("key")},
			},
		}
		client, err := NewClient(httpCtx, appCtx)
		require.NoError(t, err)
		teams, err := client.ListTeams()
		require.NoError(t, err)
		require.Len(t, teams, 1)
		assert.Equal(t, "t1", teams[0].ID)
		assert.Equal(t, "Eng", teams[0].Name)
	})
}
