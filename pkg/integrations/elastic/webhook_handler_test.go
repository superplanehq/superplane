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

func Test__ElasticWebhookHandler__Setup__AttachesConnectorToRule(t *testing.T) {
	handler := &ElasticWebhookHandler{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"rule-123","name":"High error rate"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"connector-123","name":"SuperPlane Alert - High error rate"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"rule-123",
					"name":"High error rate",
					"consumer":"alerts",
					"enabled":true,
					"params":{"index":["logs-*"]},
					"rule_type_id":".index-threshold",
					"schedule":{"interval":"1m"},
					"tags":["prod"],
					"actions":[]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
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
		URL: "https://superplane.example.com/webhooks/elastic",
		Configuration: map[string]any{
			"kibanaUrl": "https://kibana.example.com",
			"ruleId":    "rule-123",
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-123", RuleID: "rule-123"}, metadata)
	assert.NotEmpty(t, webhookCtx.Secret)

	require.Len(t, httpCtx.Requests, 4)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Equal(t, "/api/alerting/rule/rule-123", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.Equal(t, "/api/actions/connector", httpCtx.Requests[1].URL.Path)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
	assert.Equal(t, "/api/alerting/rule/rule-123", httpCtx.Requests[2].URL.Path)
	assert.Equal(t, http.MethodPut, httpCtx.Requests[3].Method)
	assert.Equal(t, "/api/alerting/rule/rule-123", httpCtx.Requests[3].URL.Path)

	body, err := io.ReadAll(httpCtx.Requests[3].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"id":"connector-123"`)
	assert.Contains(t, string(body), `"group":"default"`)
	assert.Contains(t, string(body), `"notify_when":"onActionGroupChange"`)
	assert.Contains(t, string(body), `"body":"`)
	assert.Contains(t, string(body), `{{rule.id}}`)
}
