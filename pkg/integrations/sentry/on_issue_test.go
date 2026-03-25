package sentry

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}
	t.Run("valid project persists subscription and project metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "backend",
				"actions": []string{"created"},
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)

		metadata, ok := metadataCtx.Metadata.(OnIssueMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.AppSubscriptionID)
		require.NotNil(t, metadata.Project)
		assert.Equal(t, "backend", metadata.Project.Slug)
	})

	t.Run("invalid project does not create a subscription", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectSummary{
					{ID: "1", Slug: "backend", Name: "Backend"},
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"project": "missing",
				"actions": []string{"created"},
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.ErrorContains(t, err, `project "missing" was not found`)
		assert.Empty(t, integrationCtx.Subscriptions)
		assert.Nil(t, metadataCtx.Metadata)
	})
}

func Test__OnIssue__OnIntegrationMessage(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	message := WebhookMessage{
		Resource: "issue",
		Action:   "resolved",
		Data: map[string]any{
			"issue": map[string]any{
				"id":    "123",
				"title": "Broken deploy",
				"project": map[string]any{
					"slug": "backend",
				},
			},
		},
	}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: message,
		Configuration: map[string]any{
			"project": "backend",
			"actions": []string{"resolved"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)
	assert.Equal(t, "sentry.issue", eventCtx.Payloads[0].Type)
}

func Test__OnIssue__OnIntegrationMessage__UsesTopLevelWebhookTimestamp(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource:  "issue",
			Action:    "resolved",
			Timestamp: "2026-03-24T10:15:00Z",
			Data: map[string]any{
				"issue": map[string]any{
					"id":       "123",
					"title":    "Broken deploy",
					"lastSeen": "2026-03-20T09:00:00Z",
				},
			},
		},
		Configuration: map[string]any{
			"actions": []string{"resolved"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)

	payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2026-03-24T10:15:00Z", payload["timestamp"])
}

func Test__OnIssue__OnIntegrationMessage__ArchivedAction(t *testing.T) {
	trigger := &OnIssue{}
	eventCtx := &contexts.EventContext{}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: WebhookMessage{
			Resource: "issue",
			Action:   "archived",
			Data: map[string]any{
				"issue": map[string]any{
					"id": "123",
				},
			},
		},
		Configuration: map[string]any{
			"actions": []string{"archived"},
		},
		Events: eventCtx,
		Logger: logrus.NewEntry(logrus.New()),
	})

	require.NoError(t, err)
	require.Len(t, eventCtx.Payloads, 1)
	assert.Equal(t, "sentry.issue", eventCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}
