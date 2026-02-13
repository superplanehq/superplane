package newrelic

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("valid configuration -> success", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
				"states":     []string{"ACTIVATED"},
			},
			Integration: &contexts.IntegrationContext{},
			Webhook:     &contexts.WebhookContext{},
			Metadata:    metadataCtx,
			Logger:      log.NewEntry(log.New()),
		}

		err := trigger.Setup(ctx)
		require.NoError(t, err)

		// Verify metadata was set with a URL and manual flag
		metadata, ok := metadataCtx.Metadata.(OnIssueMetadata)
		require.True(t, ok, "metadata should be OnIssueMetadata")
		assert.NotEmpty(t, metadata.URL, "webhook URL should be set in metadata")
		assert.True(t, metadata.Manual, "manual flag should be true in metadata")
	})

	t.Run("idempotent when metadata already has URL", func(t *testing.T) {
		existingURL := "http://localhost:3000/api/v1/webhooks/existing-id"
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{"url": existingURL},
		}
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      metadataCtx,
			Logger:        log.NewEntry(log.New()),
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
			"priority":  "CRITICAL",
			"state":     "ACTIVATED",
			"issue_id":  "123",
			"issue_url": "http://example.com/issue/123",
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
		assert.Equal(t, "123", data["issueId"])
	})
	t.Run("garbage payload -> 200 OK (no error)", func(t *testing.T) {
		ctx := core.WebhookRequestContext{
			Configuration: map[string]any{},
			Body:          []byte(`{"test": "notification"}`), // Simulate New Relic Test Notification
			Events:        &contexts.EventContext{},
		}

		status, err := trigger.HandleWebhook(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		// No event emitted
		assert.Equal(t, 0, ctx.Events.(*contexts.EventContext).Count())
	})

	t.Run("empty body -> 200 OK (ping)", func(t *testing.T) {
		ctx := core.WebhookRequestContext{
			Configuration: map[string]any{},
			Body:          []byte(""),
			Events:        &contexts.EventContext{},
		}

		status, err := trigger.HandleWebhook(ctx)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		// No event emitted
		assert.Equal(t, 0, ctx.Events.(*contexts.EventContext).Count())
	})
}
