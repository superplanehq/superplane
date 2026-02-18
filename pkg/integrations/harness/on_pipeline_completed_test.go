package harness

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPipelineCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("valid pipeline success event -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"eventData": {
				"accountIdentifier": "abc123",
				"orgIdentifier": "default",
				"projectIdentifier": "my_project",
				"pipelineIdentifier": "my_pipeline",
				"pipelineName": "My Pipeline",
				"planExecutionId": "exec-123",
				"executionUrl": "https://app.harness.io/...",
				"eventType": "PipelineSuccess",
				"nodeStatus": "completed"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "harness.pipeline.completed", eventContext.Payloads[0].Type)
	})

	t.Run("valid pipeline failed event -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"eventData": {
				"accountIdentifier": "abc123",
				"orgIdentifier": "default",
				"projectIdentifier": "my_project",
				"pipelineIdentifier": "my_pipeline",
				"pipelineName": "My Pipeline",
				"planExecutionId": "exec-456",
				"executionUrl": "https://app.harness.io/...",
				"eventType": "PipelineFailed",
				"nodeStatus": "completed"
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "harness.pipeline.completed", eventContext.Payloads[0].Type)
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte(`invalid json`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("missing eventData -> 400", func(t *testing.T) {
		body := []byte(`{"someOtherField": "value"}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "eventData missing")
	})

	t.Run("missing eventType -> 400", func(t *testing.T) {
		body := []byte(`{"eventData": {"pipelineName": "test"}}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "eventType missing")
	})
}

func Test__OnPipelineCompleted__Setup(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("setup generates webhook URL and stores in metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		webhookCtx := &contexts.WebhookContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: map[string]any{},
			Webhook:       webhookCtx,
		})

		assert.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(OnPipelineCompletedMetadata)
		assert.True(t, ok)
		assert.NotEmpty(t, metadata.WebhookURL)
	})

	t.Run("setup skips if webhook URL already exists", func(t *testing.T) {
		existingURL := "https://example.com/webhook/existing"
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"webhookUrl": existingURL,
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      metadataCtx,
			Configuration: map[string]any{},
		})

		assert.NoError(t, err)
	})
}
