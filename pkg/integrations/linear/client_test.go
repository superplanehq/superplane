package linear

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}
		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiToken")
	})

	t.Run("successful client creation", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-key"},
		}
		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, "test-key", client.apiKey)
	})
}

func Test__Client__GetViewer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		resp := `{"data":{"viewer":{"id":"u1","name":"Alice","email":"a@b.com"}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(resp))},
			},
		}
		appCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
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
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(resp))},
			},
		}
		appCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		client, err := NewClient(httpCtx, appCtx)
		require.NoError(t, err)
		teams, err := client.ListTeams()
		require.NoError(t, err)
		require.Len(t, teams, 1)
		assert.Equal(t, "t1", teams[0].ID)
		assert.Equal(t, "Eng", teams[0].Name)
	})
}
