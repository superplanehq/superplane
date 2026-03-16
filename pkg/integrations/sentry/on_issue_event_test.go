package sentry

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssueEvent__OnIntegrationMessage(t *testing.T) {
	trigger := &OnIssueEvent{}

	t.Run("missing event type -> ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{},
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{"issue.created"},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action in list and matching project -> event is emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "created",
				"data": map[string]any{
					"issue": map[string]any{
						"id": "123",
						"project": map[string]any{
							"slug": "test-project",
						},
					},
				},
			},
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{"issue.created"},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("action in list but different project -> event is not emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "created",
				"data": map[string]any{
					"issue": map[string]any{
						"id": "123",
						"project": map[string]any{
							"slug": "other-project",
						},
					},
				},
			},
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{"issue.created"},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action not in list -> event is not emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "resolved",
				"data": map[string]any{
					"issue": map[string]any{
						"id": "123",
						"project": map[string]any{
							"slug": "test-project",
						},
					},
				},
			},
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{"issue.created"},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("empty event types filter -> all events for matching project are emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "assigned",
				"data": map[string]any{
					"issue": map[string]any{
						"id": "123",
						"project": map[string]any{
							"slug": "test-project",
						},
					},
				},
			},
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{},
			},
			Events: eventContext,
			Logger: logrus.NewEntry(logrus.New()),
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnIssueEvent__HandleWebhook(t *testing.T) {
	trigger := &OnIssueEvent{}

	t.Run("returns 200 OK (no-op)", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})
}

func Test__OnIssueEvent__Setup(t *testing.T) {
	trigger := OnIssueEvent{}

	t.Run("project is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"project": ""},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("successful setup subscribes to integration", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		metadataCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project":    "test-project",
				"eventTypes": []string{"issue.created"},
			},
		})

		require.NoError(t, err)
		// Verify that Subscribe was called
		require.Equal(t, 1, len(integrationCtx.Subscriptions))
	})

	t.Run("already subscribed -> updates metadata only", func(t *testing.T) {
		existingSubscriptionID := uuid.New().String()
		integrationCtx := &contexts.IntegrationContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"subscriptionId": existingSubscriptionID,
				"project":        "old-project",
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"project":    "new-project",
				"eventTypes": []string{"issue.resolved"},
			},
		})

		require.NoError(t, err)
		// Verify that Subscribe was NOT called again
		require.Equal(t, 0, len(integrationCtx.Subscriptions))
	})
}

func Test__OnIssueEvent__Name(t *testing.T) {
	trigger := &OnIssueEvent{}
	assert.Equal(t, "sentry.onIssueEvent", trigger.Name())
}

func Test__OnIssueEvent__Label(t *testing.T) {
	trigger := &OnIssueEvent{}
	assert.Equal(t, "On Issue Event", trigger.Label())
}

func Test__OnIssueEvent__Description(t *testing.T) {
	trigger := &OnIssueEvent{}
	assert.Equal(t, "Listen to Sentry issue events", trigger.Description())
}

func Test__OnIssueEvent__Icon(t *testing.T) {
	trigger := &OnIssueEvent{}
	assert.Equal(t, "sentry", trigger.Icon())
}

func Test__OnIssueEvent__Color(t *testing.T) {
	trigger := &OnIssueEvent{}
	assert.Equal(t, "purple", trigger.Color())
}

func Test__OnIssueEvent__Configuration(t *testing.T) {
	trigger := &OnIssueEvent{}
	config := trigger.Configuration()
	assert.Len(t, config, 2)

	// Check project field
	assert.Equal(t, "project", config[0].Name)
	assert.True(t, config[0].Required)

	// Check eventTypes field
	assert.Equal(t, "eventTypes", config[1].Name)
	assert.False(t, config[1].Required)
}
