package logfire

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type failingEventContext struct{}

func (f *failingEventContext) Emit(string, any) error {
	return errors.New("emit failed")
}

func TestOnAlertReceived_Setup_RequestsStableWebhookConfiguration(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "lf_api_us_123"},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","organization_name":"acme","project_name":"backend"}]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"alt_456","name":"Latency spike"}]`)),
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		HTTP:        httpCtx,
		Metadata:    metadataCtx,
		Configuration: map[string]any{
			"project": "proj_123",
			"alert":   "alt_456",
		},
	})
	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)

	requestedConfig, ok := integrationCtx.WebhookRequests[0].(onAlertReceivedWebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, onAlertReceivedEventType, requestedConfig.EventType)
	assert.Equal(t, onAlertReceivedResource, requestedConfig.Resource)
	assert.Equal(t, "proj_123", requestedConfig.ProjectID)
	assert.Equal(t, "alt_456", requestedConfig.AlertID)

	metadata, ok := metadataCtx.Get().(OnAlertReceivedNodeMetadata)
	require.True(t, ok)
	require.NotNil(t, metadata.Project)
	assert.Equal(t, "backend", metadata.Project.Name)
	require.NotNil(t, metadata.Alert)
	assert.Equal(t, "Latency spike", metadata.Alert.Name)
}

func TestOnAlertReceived_Setup_MissingProjectId(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}
	err := trigger.Setup(core.TriggerContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "lf_api_us_123"},
		},
		HTTP:     &contexts.HTTPContext{},
		Metadata: &contexts.MetadataContext{},
		Configuration: map[string]any{
			"project": "",
			"alert":   "alt_456",
		},
	})
	require.ErrorContains(t, err, "project is required")
}

func TestOnAlertReceived_Setup_MissingAlertId(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}
	err := trigger.Setup(core.TriggerContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "lf_api_us_123"},
		},
		HTTP:     &contexts.HTTPContext{},
		Metadata: &contexts.MetadataContext{},
		Configuration: map[string]any{
			"project": "proj_123",
			"alert":   "",
		},
	})
	require.ErrorContains(t, err, "alert is required")
}

func TestOnAlertReceived_Setup_InvalidProject(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_other","organization_name":"acme","project_name":"other"}]`)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "lf_api_us_123"},
		},
		HTTP:     httpCtx,
		Metadata: &contexts.MetadataContext{},
		Configuration: map[string]any{
			"project": "proj_123",
			"alert":   "alt_456",
		},
	})
	require.ErrorContains(t, err, "invalid Logfire project selection")
}

func TestOnAlertReceived_Setup_InvalidAlert(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","organization_name":"acme","project_name":"backend"}]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"alt_other","name":"Other alert"}]`)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "lf_api_us_123"},
		},
		HTTP:     httpCtx,
		Metadata: &contexts.MetadataContext{},
		Configuration: map[string]any{
			"project": "proj_123",
			"alert":   "alt_456",
		},
	})
	require.ErrorContains(t, err, "invalid Logfire alert selection")
}

func TestOnAlertReceived_HandleWebhook(t *testing.T) {
	t.Parallel()

	trigger := &OnAlertReceived{}

	t.Run("invalid body returns bad request", func(t *testing.T) {
		t.Parallel()

		status, response, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   []byte("{"),
			Events: &contexts.EventContext{},
		})

		require.ErrorContains(t, err, "failed to parse request body")
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Nil(t, response)
	})

	t.Run("emit failure returns internal server error", func(t *testing.T) {
		t.Parallel()

		status, response, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   []byte(`{"alert_id":"alt_123"}`),
			Events: &failingEventContext{},
		})

		require.ErrorContains(t, err, "failed to emit alert event")
		assert.Equal(t, http.StatusInternalServerError, status)
		assert.Nil(t, response)
	})

	t.Run("success emits normalized payload", func(t *testing.T) {
		t.Parallel()

		events := &contexts.EventContext{}
		status, response, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"alert_id":"alt_123",
				"alert_name":"High latency",
				"event_type":"firing",
				"data":{
					"text":":warning: <https://logfire-us.pydantic.dev/alerts/alt_123|High latency>",
					"attachments":[{"text":"p95 latency exceeded threshold","ts":1711200000}]
				}
			}`),
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Nil(t, response)
		require.Len(t, events.Payloads, 1)

		payload := events.Payloads[0]
		assert.Equal(t, "logfire.alert.received", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "alt_123", data["alertId"])
		assert.Equal(t, "High latency", data["alertName"])
		assert.Equal(t, "firing", data["eventType"])
		assert.Equal(t, "warning", data["severity"])
		assert.Equal(t, "p95 latency exceeded threshold", data["message"])
	})
}
