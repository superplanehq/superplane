package dash0

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__buildNotificationChannelDefinition(t *testing.T) {
	def := buildNotificationChannelDefinition("SuperPlane (test)", "https://hooks.example.com/webhook")

	assert.Equal(t, "Dash0NotificationChannel", def.Kind)
	assert.Equal(t, "SuperPlane (test)", def.Metadata.Name)
	assert.Equal(t, "webhook", def.Spec.Type)
	assert.Equal(t, "https://hooks.example.com/webhook", def.Spec.Config.URL)
	require.NotNil(t, def.Spec.Routing)
	assert.Empty(t, def.Spec.Routing.Assets)
	require.Len(t, def.Spec.Routing.Filters, 1)
	require.Len(t, def.Spec.Routing.Filters[0], 1)
	assert.Equal(t, "dash0.failed_check.max_status", def.Spec.Routing.Filters[0][0].Key)
	assert.Equal(t, "is_any", def.Spec.Routing.Filters[0][0].Operator)
}

func Test__Client__CreateNotificationChannel(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"kind": "Dash0NotificationChannel",
					"metadata": {
						"name": "SuperPlane (test)",
						"labels": { "dash0.com/id": "channel-uuid-123" }
					},
					"spec": { "type": "webhook", "config": { "url": "https://hooks.example.com/webhook" } }
				}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "token123",
			"baseURL":  "https://api.us-west-2.aws.dash0.com",
		},
	}

	client, err := NewClient(httpContext, integrationCtx)
	require.NoError(t, err)

	id, err := client.CreateNotificationChannel("SuperPlane (test)", "https://hooks.example.com/webhook")
	require.NoError(t, err)
	assert.Equal(t, "channel-uuid-123", id)

	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/notification-channels")

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var sent NotificationChannelDefinition
	require.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, "is_any", sent.Spec.Routing.Filters[0][0].Operator)
}

func Test__Client__ListNotificationChannels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{
						"kind": "Dash0NotificationChannel",
						"metadata": {
							"name": "SuperPlane (abc)",
							"labels": { "dash0.com/id": "id-1" }
						},
						"spec": { "type": "webhook", "config": { "url": "https://example.com" } }
					}
				]`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "token123",
			"baseURL":  "https://api.us-west-2.aws.dash0.com",
		},
	}

	client, err := NewClient(httpContext, integrationCtx)
	require.NoError(t, err)

	channels, err := client.ListNotificationChannels()
	require.NoError(t, err)
	require.Len(t, channels, 1)
	assert.Equal(t, "id-1", channels[0].ID)
	assert.Equal(t, "SuperPlane (abc)", channels[0].Name)
}

func Test__Client__UpdateNotificationChannel(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "token123",
			"baseURL":  "https://api.us-west-2.aws.dash0.com",
		},
	}

	client, err := NewClient(httpContext, integrationCtx)
	require.NoError(t, err)

	err = client.UpdateNotificationChannel("channel-uuid-123", "SuperPlane (test)", "https://hooks.example.com/new")
	require.NoError(t, err)

	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/notification-channels/channel-uuid-123")
}

func Test__Client__DeleteNotificationChannel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		err = client.DeleteNotificationChannel("channel-uuid-123")
		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("404 -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		err = client.DeleteNotificationChannel("missing-channel")
		require.NoError(t, err)
	})
}

func Test__provisionNotificationChannel(t *testing.T) {
	integrationID := uuid.MustParse("8f5fbc57-2738-409a-a6f8-af65c2de733c")
	webhookURL := "https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/webhook"
	channelName := notificationChannelName(integrationID)

	t.Run("creates channel when none exists", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"kind": "Dash0NotificationChannel",
						"metadata": {
							"name": "` + channelName + `",
							"labels": { "dash0.com/id": "new-channel-id" }
						},
						"spec": { "type": "webhook", "config": { "url": "` + webhookURL + `" } }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		id, err := provisionNotificationChannel(client, integrationCtx, webhookURL)
		require.NoError(t, err)
		assert.Equal(t, "new-channel-id", id)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
	})

	t.Run("reuses channel id from metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: integrationID.String(),
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
			Metadata: Metadata{
				NotificationChannelID: "existing-channel-id",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		id, err := provisionNotificationChannel(client, integrationCtx, webhookURL)
		require.NoError(t, err)
		assert.Equal(t, "existing-channel-id", id)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	})
}
