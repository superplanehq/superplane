package slack

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Slack__ListResources(t *testing.T) {
	s := &Slack{}

	t.Run("unknown resource type returns empty", func(t *testing.T) {
		resources, err := s.ListResources("unknown", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"botToken": "token-123",
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("lists Slack channels", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.list", req.URL.String())
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "Bearer token-123", req.Header.Get("Authorization"))

			return jsonResponse(http.StatusOK, `{
				"ok": true,
				"channels": [
					{"id": "C123", "name": "engineering"},
					{"id": "C456", "name": "deployments"}
				]
			}`), nil
		})

		resources, err := s.ListResources("channel", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"botToken": "token-123",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, core.IntegrationResource{Type: "channel", Name: "engineering", ID: "C123"}, resources[0])
		assert.Equal(t, core.IntegrationResource{Type: "channel", Name: "deployments", ID: "C456"}, resources[1])
	})

	t.Run("Slack API error returns error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/conversations.list", req.URL.String())
			return jsonResponse(http.StatusOK, `{"ok": false, "error": "invalid_auth"}`), nil
		})

		resources, err := s.ListResources("channel", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"botToken": "token-123",
				},
			},
		})

		require.ErrorContains(t, err, "failed to list channels: invalid_auth")
		assert.Nil(t, resources)
	})
}
