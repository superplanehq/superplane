package railway

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnDeploymentEvent__HandleWebhook(t *testing.T) {
	trigger := &OnDeploymentEvent{}

	t.Run("invalid JSON payload -> 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{invalid json`),
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "failed to parse webhook payload")
	})

	t.Run("non-deployment event is ignored", func(t *testing.T) {
		body := []byte(`{"type": "OTHER_EVENT"}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("deployment event with no status filter -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.succeeded",
			"details": {
				"status": "SUCCESS"
			},
			"resource": {
				"deployment": {
					"id": "deploy-123"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("deployment event matching status filter -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.succeeded",
			"details": {
				"status": "SUCCESS"
			},
			"resource": {
				"deployment": {
					"id": "deploy-123"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{"succeeded", "failed"},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("deployment event not matching status filter -> event is not emitted", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.building",
			"details": {
				"status": "BUILDING"
			},
			"resource": {
				"deployment": {
					"id": "deploy-123"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{"succeeded", "failed"},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("deployment event with failed status -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.failed",
			"details": {
				"status": "FAILED"
			},
			"resource": {
				"deployment": {
					"id": "deploy-456"
				},
				"service": {
					"id": "srv-123",
					"name": "api-server"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{"failed"},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("deployment event with crashed status -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.crashed",
			"details": {
				"status": "CRASHED"
			},
			"resource": {
				"deployment": {
					"id": "deploy-789"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{"crashed"},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("deployment event without action in type -> event is emitted when no filter", func(t *testing.T) {
		body := []byte(`{
			"type": "Deployment.",
			"resource": {
				"deployment": {
					"id": "deploy-123"
				}
			}
		}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: http.Header{},
			Configuration: map[string]any{
				"project":  "proj-123",
				"statuses": []string{},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnDeploymentEvent__Setup(t *testing.T) {
	trigger := OnDeploymentEvent{}

	t.Run("project is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": ""},
		})

		require.ErrorContains(t, err, "project is required")
	})
}

func Test__ExtractEventAction(t *testing.T) {
	t.Run("extracts action from valid event type", func(t *testing.T) {
		assert.Equal(t, "succeeded", extractEventAction("Deployment.succeeded"))
	})

	t.Run("extracts action from failed event type", func(t *testing.T) {
		assert.Equal(t, "failed", extractEventAction("Deployment.failed"))
	})

	t.Run("extracts action from crashed event type", func(t *testing.T) {
		assert.Equal(t, "crashed", extractEventAction("Deployment.crashed"))
	})

	t.Run("returns empty string for event type without dot", func(t *testing.T) {
		assert.Equal(t, "", extractEventAction("Deployment"))
	})

	t.Run("returns empty string for empty event type", func(t *testing.T) {
		assert.Equal(t, "", extractEventAction(""))
	})

	t.Run("handles event type with empty action", func(t *testing.T) {
		assert.Equal(t, "", extractEventAction("Deployment."))
	})

	t.Run("handles event type with multiple dots", func(t *testing.T) {
		assert.Equal(t, "succeeded.extra", extractEventAction("Deployment.succeeded.extra"))
	})
}
