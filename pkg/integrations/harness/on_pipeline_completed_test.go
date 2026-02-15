package harness

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPipelineCompleted__Setup(t *testing.T) {
	trigger := &OnPipelineCompleted{}
	metadataCtx := &contexts.MetadataContext{}

	err := trigger.Setup(core.TriggerContext{
		Configuration: OnPipelineCompletedConfiguration{Statuses: []string{"failed"}},
		Metadata:      metadataCtx,
		Webhook:       &contexts.WebhookContext{},
	})

	require.NoError(t, err)
	metadata, ok := metadataCtx.Get().(OnPipelineCompletedMetadata)
	require.True(t, ok)
	assert.NotEmpty(t, metadata.WebhookURL)
}

func Test__OnPipelineCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("invalid webhook secret -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer wrong")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Events: &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "invalid webhook authorization")
	})

	t.Run("emits event when status and pipeline match", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"failed"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, OnPipelineCompletedPayloadType, events.Payloads[0].Type)
	})

	t.Run("ignores non-matching status", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"SUCCEEDED"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"failed"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignores event when configured pipeline is set but payload pipeline is missing", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-1","nodeStatus":"completed"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"succeeded"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("without webhook secret accepts request", func(t *testing.T) {
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Body:          []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-2","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"failed"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("maps alternate status values", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PIPELINE_END","data":{"planExecutionId":"exec-3","pipelineIdentifier":"deploy","status":"ERROR"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"failed"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())
	})

	t.Run("ignores non pipeline completed event types", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"STAGE_END","data":{"planExecutionId":"exec-3","pipelineIdentifier":"deploy","status":"FAILED"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"failed"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("maps completed status to succeeded for filtering and emitted payload", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer expected")
		events := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Body:    []byte(`{"eventType":"PipelineEnd","eventData":{"planExecutionId":"exec-3","pipelineIdentifier":"deploy","nodeStatus":"completed"}}`),
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"webhookSecret": "expected",
			}},
			Configuration: OnPipelineCompletedConfiguration{PipelineIdentifier: "deploy", Statuses: []string{"succeeded"}},
			Events:        events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, events.Count())

		payload, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "succeeded", payload["status"])
	})
}
