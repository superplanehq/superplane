package dash0

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertNotification__Setup(t *testing.T) {
	trigger := &OnAlertNotification{}

	t.Run("no previous subscription -> subscribes and stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integration,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		stored, ok := metadata.Metadata.(OnAlertNotificationMetadata)
		require.True(t, ok)
		require.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("subscription already exists -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnAlertNotificationMetadata{SubscriptionID: uuid.NewString()},
		}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integration,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		require.Empty(t, integration.Subscriptions)
	})
}

func Test__OnAlertNotification__OnIntegrationMessage(t *testing.T) {
	trigger := &OnAlertNotification{}
	events := &contexts.EventContext{}
	message := map[string]any{
		"type": "alert.ongoing",
		"data": map[string]any{
			"issue": map[string]any{
				"status": "critical",
			},
		},
	}

	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message:       message,
		Configuration: map[string]any{"statuses": []string{"critical", "degraded"}},
		Logger:        logrus.NewEntry(logrus.New()),
		Events:        events,
	})

	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
	assert.Equal(t, "dash0.alertNotification", events.Payloads[0].Type)

	payload, ok := events.Payloads[0].Data.(AlertNotificationData)
	require.True(t, ok)
	require.NotNil(t, payload.Issue)
	assert.Equal(t, "critical", payload.Issue.Status)
}
