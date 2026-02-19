package dash0

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnNotification__Setup(t *testing.T) {
	trigger := &OnNotification{}

	t.Run("requests webhook and stores URL in metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		metadataCtx := &contexts.MetadataContext{}
		webhookURL := "https://example.com/api/v1/webhooks/some-id"

		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Webhook:     &onNotificationWebhookContext{url: webhookURL},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		metadata, ok := metadataCtx.Metadata.(OnNotificationMetadata)
		require.True(t, ok)
		assert.Equal(t, webhookURL, metadata.WebhookURL)
	})
}

func Test__OnNotification__HandleWebhook(t *testing.T) {
	trigger := &OnNotification{}

	t.Run("invalid body returns 400", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   []byte("not-json"),
			Events: eventsCtx,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "failed to parse request body")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("valid payload emits dash0.notification event", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:   []byte(`{"type":"alert","checkName":"Login API","severity":"critical"}`),
			Events: eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 1)
		assert.Equal(t, "dash0.notification", eventsCtx.Payloads[0].Type)
		data := eventsCtx.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "alert", data["type"])
		assert.Equal(t, "Login API", data["checkName"])
		assert.Equal(t, "critical", data["severity"])
	})
}

type onNotificationWebhookContext struct {
	url string
}

func (s *onNotificationWebhookContext) GetSecret() ([]byte, error)           { return nil, nil }
func (s *onNotificationWebhookContext) ResetSecret() ([]byte, []byte, error) { return nil, nil, nil }
func (s *onNotificationWebhookContext) Setup() (string, error)               { return s.url, nil }
func (s *onNotificationWebhookContext) GetBaseURL() string                   { return "https://example.com/api/v1" }
