package dash0

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnNotification__Setup(t *testing.T) {
	trigger := &OnNotification{}

	t.Run("no previous subscription -> subscribes and stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		integration := &contexts.IntegrationContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration: integration,
			Metadata:    metadata,
		})

		require.NoError(t, err)
		require.Len(t, integration.Subscriptions, 1)

		stored, ok := metadata.Metadata.(OnNotificationMetadata)
		require.True(t, ok)
		require.NotEmpty(t, stored.SubscriptionID)
	})

	t.Run("subscription already exists -> no-op", func(t *testing.T) {
		metadata := &contexts.MetadataContext{
			Metadata: OnNotificationMetadata{SubscriptionID: uuid.NewString()},
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

func Test__OnNotification__OnIntegrationMessage(t *testing.T) {
	trigger := &OnNotification{}
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
	assert.Equal(t, "dash0.notification", events.Payloads[0].Type)

	payload, ok := events.Payloads[0].Data.(NotificationEvent)
	require.True(t, ok)
	require.NotNil(t, payload.Data.Issue)
	assert.Equal(t, "alert.ongoing", payload.Type)
	assert.Equal(t, "critical", payload.Data.Issue.Status)
}

type dash0RequestIntegrationContext struct {
	*contexts.IntegrationContext
	subscriptions []core.IntegrationSubscriptionContext
	err           error
}

func (c *dash0RequestIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return c.subscriptions, c.err
}

type dash0TestSubscription struct {
	messages []any
}

func (s *dash0TestSubscription) Configuration() any {
	return map[string]any{"type": "notification"}
}

func (s *dash0TestSubscription) SendMessage(message any) error {
	s.messages = append(s.messages, message)
	return nil
}

func Test__Dash0__HandleRequest(t *testing.T) {
	integration := &Dash0{}

	t.Run("unknown path -> 404", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/id/unknown", http.NoBody)
		response := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:  request,
			Response: response,
			Logger:   logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("non-post request -> 405", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/id/webhook", http.NoBody)
		response := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:  request,
			Response: response,
			Logger:   logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	})

	t.Run("invalid json -> 400", func(t *testing.T) {
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/integrations/id/webhook",
			bytes.NewBufferString("{invalid"),
		)
		response := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:  request,
			Response: response,
			Logger:   logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("valid webhook -> forwards message to subscriptions", func(t *testing.T) {
		subscription := &dash0TestSubscription{}
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/integrations/id/webhook",
			bytes.NewBufferString(`{"notification":{"id":"n1","severity":"critical"}}`),
		)
		response := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Request:  request,
			Response: response,
			Logger:   logrus.NewEntry(logrus.New()),
			Integration: &dash0RequestIntegrationContext{
				IntegrationContext: &contexts.IntegrationContext{},
				subscriptions:      []core.IntegrationSubscriptionContext{subscription},
			},
		})

		assert.Equal(t, http.StatusOK, response.Code)
		require.Len(t, subscription.messages, 1)
		payload, ok := subscription.messages[0].(map[string]any)
		require.True(t, ok)
		notification, ok := payload["notification"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "critical", notification["severity"])
	})
}
