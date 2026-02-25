package newrelic

import (
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

	t.Run("valid configuration -> requests webhook", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-TEST",
				"site":       "US",
			},
		}

		ctx := core.TriggerContext{
			Configuration: map[string]any{
				"account":    "123456",
				"priorities": []string{"CRITICAL"},
				"states":     []string{"ACTIVATED"},
			},
			Integration: integrationCtx,
			Webhook:     &contexts.WebhookContext{},
			Metadata:    metadataCtx,
			Logger:      log.NewEntry(log.New()),
		}

		err := trigger.Setup(ctx)
		require.NoError(t, err)

		// Verify RequestWebhook was called with the correct configuration
		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok, "should be WebhookConfiguration")
		assert.Equal(t, "123456", webhookConfig.Account)
	})

	t.Run("missing account -> error", func(t *testing.T) {
		ctx := core.TriggerContext{
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "NRAK-TEST",
					"site":       "US",
				},
			},
			Logger: log.NewEntry(log.New()),
		}

		err := trigger.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is required")
	})
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("missing auth header -> 401 Unauthorized", func(t *testing.T) {
		ctx := core.WebhookRequestContext{
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Headers: http.Header{},
			Body:    []byte(`{}`),
		}

		status, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("invalid secret -> 401 Unauthorized", func(t *testing.T) {
		ctx := core.WebhookRequestContext{
			Webhook: &contexts.WebhookContext{Secret: "valid-secret"},
			Headers: http.Header{"X-Superplane-Secret": []string{"invalid-secret"}},
			Body:    []byte(`{}`),
		}

		status, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("valid payload -> emits event", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		ctx := core.WebhookRequestContext{
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Headers: http.Header{"X-Superplane-Secret": []string{"test-secret"}},
			Body: []byte(`{
				"issue_id": "issue-123",
				"title": "Test Issue",
				"priority": "CRITICAL",
				"state": "ACTIVATED",
				"issue_url": "https://newrelic.com/issue/123"
			}`),
			Events: eventCtx,
			Configuration: map[string]any{
				"account": "123456",
			},
		}

		status, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)

		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "newrelic.issue_activated", eventCtx.Payloads[0].Type)

		data := eventCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "issue-123", data["issueId"])
		assert.Equal(t, "CRITICAL", data["priority"])
	})

	t.Run("filtered priority -> no event", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		ctx := core.WebhookRequestContext{
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Headers: http.Header{"X-Superplane-Secret": []string{"test-secret"}},
			Body: []byte(`{
				"issue_id": "issue-123",
				"priority": "LOW",
				"state": "ACTIVATED"
			}`),
			Events: eventCtx,
			Configuration: map[string]any{
				"priorities": []string{"CRITICAL"},
			},
		}

		status, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})
}
