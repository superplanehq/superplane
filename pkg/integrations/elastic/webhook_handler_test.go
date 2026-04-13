package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ElasticWebhookHandler__Setup__CreatesConnector(t *testing.T) {
	handler := &ElasticWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"connector-123","name":"SuperPlane Alert"}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}
	webhookCtx := &contexts.WebhookContext{
		URL: "https://superplane.example.com/api/v1/webhooks/elastic-123",
		Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-123"}, metadata)
	assert.NotEmpty(t, webhookCtx.Secret)

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Equal(t, "/api/actions/connectors", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.Equal(t, "/api/actions/connector", httpCtx.Requests[1].URL.Path)
}

func Test__ElasticWebhookHandler__Setup__ReusesExistingConnector(t *testing.T) {
	handler := &ElasticWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{
						"id":"connector-456",
						"name":"SuperPlane Alert",
						"connector_type_id":".webhook",
						"config":{
							"url":"https://superplane.example.com/api/v1/webhooks/elastic-123",
							"method":"post",
							"headers":{"X-Superplane-Secret":"existing-secret"}
						}
					}
				]`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"apiKey":    "api-key",
		},
	}
	webhookCtx := &contexts.WebhookContext{
		URL: "https://superplane.example.com/api/v1/webhooks/elastic-123",
		Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-456"}, metadata)
	assert.Equal(t, []byte("existing-secret"), webhookCtx.Secret)

	// Only one request: GET connectors (connector already exists, no create needed)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Equal(t, "/api/actions/connectors", httpCtx.Requests[0].URL.Path)
}
