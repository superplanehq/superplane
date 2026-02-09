package newrelic

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("valid configuration -> success", func(t *testing.T) {
		ctx := core.TriggerContext{
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
				"states":     []string{"ACTIVATED"},
			},
			Integration: &contexts.IntegrationContext{},
		}

		err := trigger.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("filters out non-matching priority", func(t *testing.T) {
		payload := map[string]any{
			"issue_id": "123",
			"priority": "LOW",
			"state":    "ACTIVATED",
		}
		body, _ := json.Marshal(payload)

		ctx := core.WebhookRequestContext{
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
			},
			Body:   body,
			Events: &contexts.EventContext{},
		}

		status, err := trigger.HandleWebhook(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		// No event emitted
		assert.Equal(t, 0, ctx.Events.(*contexts.EventContext).Count())
	})

	t.Run("matches and emits event", func(t *testing.T) {
		payload := map[string]any{
			"priority": "CRITICAL",
			"state":    "ACTIVATED",
			"issue_id": "123",
		}
		body, _ := json.Marshal(payload)

		ctx := core.WebhookRequestContext{
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
			},
			Body:   body,
			Events: &contexts.EventContext{},
		}

		status, err := trigger.HandleWebhook(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		eventCtx := ctx.Events.(*contexts.EventContext)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "newrelic.issue_activated", eventCtx.Payloads[0].Type)
		
		data, ok := eventCtx.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "123", data["issue_id"])
	})
}
