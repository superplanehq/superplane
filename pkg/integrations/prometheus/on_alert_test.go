package prometheus

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlert__Setup(t *testing.T) {
	trigger := &OnAlert{}

	t.Run("at least one status is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"statuses": []string{}},
			Integration:   &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "at least one status")
	})

	t.Run("valid setup requests shared webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{"statuses": []string{AlertStateFiring}},
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		assert.IsType(t, struct{}{}, integrationCtx.WebhookRequests[0])
	})
}

func Test__OnAlert__HandleWebhook(t *testing.T) {
	trigger := &OnAlert{}
	payload := []byte(`
	{
	  "status":"firing",
	  "receiver":"superplane",
	  "groupKey":"{}:{alertname=\"HighRequestLatency\"}",
	  "groupLabels":{"alertname":"HighRequestLatency"},
	  "commonLabels":{"alertname":"HighRequestLatency","severity":"critical"},
	  "commonAnnotations":{"summary":"API latency above threshold"},
	  "externalURL":"http://alertmanager.example.com",
	  "alerts":[
	    {
	      "status":"firing",
	      "labels":{"alertname":"HighRequestLatency","instance":"api-1:9090","job":"api"},
	      "annotations":{"summary":"API latency above threshold","description":"P95 latency above 500ms"},
	      "startsAt":"2026-01-19T12:00:00Z",
	      "endsAt":"0001-01-01T00:00:00Z",
	      "generatorURL":"http://prometheus.example.com/graph?g0.expr=...",
	      "fingerprint":"abc123"
	    },
	    {
	      "status":"resolved",
	      "labels":{"alertname":"DiskAlmostFull","instance":"node-1:9100","job":"node"},
	      "annotations":{"summary":"Disk recovered"},
	      "startsAt":"2026-01-19T10:00:00Z",
	      "endsAt":"2026-01-19T12:10:00Z",
	      "generatorURL":"http://prometheus.example.com/graph?g0.expr=...",
	      "fingerprint":"def456"
	    }
	  ]
	}
	`)

	t.Run("missing bearer auth returns 403", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		secret := mustSecret(t, webhookRuntimeConfiguration{AuthType: AuthTypeBearer, BearerToken: "token-1"})

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{AlertStateFiring}},
			Webhook:       &contexts.WebhookContext{Secret: string(secret)},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusForbidden, code)
		require.ErrorContains(t, err, "missing bearer authorization")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		secret := mustSecret(t, webhookRuntimeConfiguration{AuthType: AuthTypeNone})

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte("not-json"),
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{AlertStateFiring}},
			Webhook:       &contexts.WebhookContext{Secret: string(secret)},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "failed to parse request body")
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("status filtered out returns 200 and no events", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		secret := mustSecret(t, webhookRuntimeConfiguration{AuthType: AuthTypeNone})

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       http.Header{},
			Configuration: map[string]any{"statuses": []string{AlertStateResolved}, "alertNames": []string{"OnlyOther"}},
			Webhook:       &contexts.WebhookContext{Secret: string(secret)},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Len(t, eventsCtx.Payloads, 0)
	})

	t.Run("valid firing and resolved alerts are emitted", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		secret := mustSecret(t, webhookRuntimeConfiguration{AuthType: AuthTypeBasic, Username: "user", Password: "pass"})
		headers := http.Header{}
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          payload,
			Headers:       headers,
			Configuration: map[string]any{"statuses": []string{AlertStateFiring, AlertStateResolved}},
			Webhook:       &contexts.WebhookContext{Secret: string(secret)},
			Events:        eventsCtx,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Len(t, eventsCtx.Payloads, 2)
		assert.Equal(t, PrometheusAlertPayloadType, eventsCtx.Payloads[0].Type)
		assert.Equal(t, "HighRequestLatency", eventsCtx.Payloads[0].Data.(map[string]any)["labels"].(map[string]string)["alertname"])
		assert.Equal(t, "resolved", eventsCtx.Payloads[1].Data.(map[string]any)["status"])
	})
}

func mustSecret(t *testing.T, config webhookRuntimeConfiguration) []byte {
	t.Helper()
	raw, err := json.Marshal(config)
	require.NoError(t, err)
	return raw
}
