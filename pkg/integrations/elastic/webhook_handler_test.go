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

func Test__ElasticWebhookHandler__Setup__CreatesConnectorWithoutRule(t *testing.T) {
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
			"ruleId":    "",
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-123", RuleID: ""}, metadata)
	assert.NotEmpty(t, webhookCtx.Secret)

	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Equal(t, "/api/actions/connectors", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[1].Method)
	assert.Equal(t, "/api/actions/connector", httpCtx.Requests[1].URL.Path)
}

func Test__ElasticWebhookHandler__Setup__ReusesExistingConnectorAndAttachesSelectedRule(t *testing.T) {
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
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"rule-123",
					"name":"High error rate",
					"consumer":"alerts",
					"params":{"index":["logs-*"]},
					"rule_type_id":"xpack.synthetics.alerts.monitorStatus",
					"schedule":{"interval":"1m"},
					"tags":["prod"],
					"actions":[],
					"alert_delay":{"active":1}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"xpack.synthetics.alerts.monitorStatus","default_action_group_id":"xpack.synthetics.alerts.actionGroups.monitorStatus"}
				]`)),
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
		URL: "https://superplane.example.com/api/v1/webhooks/elastic-123",
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
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-456", RuleID: "rule-123"}, metadata)
	assert.Equal(t, []byte("existing-secret"), webhookCtx.Secret)

	require.Len(t, httpCtx.Requests, 4)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Equal(t, "/api/actions/connectors", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[1].Method)
	assert.Equal(t, "/api/alerting/rule/rule-123", httpCtx.Requests[1].URL.Path)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
	assert.Equal(t, "/api/alerting/rule_types", httpCtx.Requests[2].URL.Path)
	assert.Equal(t, http.MethodPut, httpCtx.Requests[3].Method)
	assert.Equal(t, "/api/alerting/rule/rule-123", httpCtx.Requests[3].URL.Path)

	body, err := io.ReadAll(httpCtx.Requests[3].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"id":"connector-456"`)
	assert.Contains(t, string(body), `"group":"xpack.synthetics.alerts.actionGroups.monitorStatus"`)
}

func Test__ElasticWebhookHandler__Setup__DetachesPreviousRuleWhenRuleChanges(t *testing.T) {
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
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"rule-123",
					"name":"New rule",
					"consumer":"alerts",
					"params":{"index":["logs-*"]},
					"rule_type_id":"xpack.synthetics.alerts.monitorStatus",
					"schedule":{"interval":"1m"},
					"tags":["prod"],
					"actions":[],
					"alert_delay":{"active":1}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"xpack.synthetics.alerts.monitorStatus","default_action_group_id":"xpack.synthetics.alerts.actionGroups.monitorStatus"}
				]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":"rule-old",
					"name":"Old rule",
					"consumer":"alerts",
					"params":{"index":["logs-*"]},
					"rule_type_id":"xpack.synthetics.alerts.monitorStatus",
					"schedule":{"interval":"1m"},
					"tags":["prod"],
					"actions":[{"id":"connector-456","group":"xpack.synthetics.alerts.actionGroups.monitorStatus","params":{"body":"test"}}],
					"alert_delay":{"active":1}
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
		URL:      "https://superplane.example.com/api/v1/webhooks/elastic-123",
		Metadata: map[string]any{"connectorId": "connector-456", "ruleId": "rule-old"},
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
	assert.Equal(t, webhookMetadata{ConnectorID: "connector-456", RuleID: "rule-123"}, metadata)

	require.Len(t, httpCtx.Requests, 6)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[4].Method)
	assert.Equal(t, "/api/alerting/rule/rule-old", httpCtx.Requests[4].URL.Path)
	assert.Equal(t, http.MethodPut, httpCtx.Requests[5].Method)
	assert.Equal(t, "/api/alerting/rule/rule-old", httpCtx.Requests[5].URL.Path)
}
