package render

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Render_OnEvent__Setup(t *testing.T) {
	trigger := &OnEvent{}
	integrationCtx := &contexts.IntegrationContext{}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		Configuration: map[string]any{
			"eventTypes": []string{"deploy_ended"},
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
}

func Test__Render_OnEvent__HandleWebhook(t *testing.T) {
	trigger := &OnEvent{}

	payload := map[string]any{
		"type":      "deploy_ended",
		"timestamp": "2026-02-05T16:00:01.000000Z",
		"data": map[string]any{
			"id":          "evj-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "succeeded",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)

	t.Run("missing signature headers -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "missing signature headers")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		invalidHeaders := buildSignedHeaders(secret, body)
		invalidHeaders.Set("webhook-signature", "v1,invalid")

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       invalidHeaders,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "invalid signature")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("event type filter does not match -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventTypes": []string{"build_ended"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("service ID filter matches -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"serviceIdFilter": []map[string]any{
					{"type": configuration.PredicateTypeEquals, "value": "srv-cukouhrtq21c73e9scng"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload, eventCtx.Payloads[0].Data)
	})

	t.Run("service name filter does not match -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"serviceNameFilter": []map[string]any{
					{"type": configuration.PredicateTypeEquals, "value": "worker"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("valid signature and matching filters -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"eventTypes": []string{"deploy_ended"},
				"serviceNameFilter": []map[string]any{
					{"type": configuration.PredicateTypeMatches, "value": "backend-.*"},
				},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload, eventCtx.Payloads[0].Data)
	})

	t.Run("multiple signatures header -> accepts matching v1 signature", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		validHeaders := buildSignedHeaders(secret, body)
		validSignature := strings.TrimPrefix(validHeaders.Get("webhook-signature"), "v1,")

		headersWithMultipleSignatures := http.Header{}
		headersWithMultipleSignatures.Set("webhook-id", validHeaders.Get("webhook-id"))
		headersWithMultipleSignatures.Set("webhook-timestamp", validHeaders.Get("webhook-timestamp"))
		headersWithMultipleSignatures.Set("webhook-signature", "v1,invalid v1,"+validSignature)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headersWithMultipleSignatures,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload, eventCtx.Payloads[0].Data)
	})

	t.Run("whsec secret format -> accepts decoded signing key", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		rawSecret := "test-secret"
		webhookSecret := "whsec_" + base64.RawStdEncoding.EncodeToString([]byte(rawSecret))
		headers := buildSignedHeaders(rawSecret, body)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: webhookSecret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload, eventCtx.Payloads[0].Data)
	})
}

func Test__renderPayloadType(t *testing.T) {
	assert.Equal(t, "render.build.ended", renderPayloadType("build_ended"))
	assert.Equal(t, "render.server.failed", renderPayloadType("server_failed"))
	assert.Equal(t, "render.autoscaling.ended", renderPayloadType("autoscaling_ended"))
	assert.Equal(t, "render.event", renderPayloadType(""))
}

func buildSignedHeaders(secret string, body []byte) http.Header {
	webhookID := "msg_2mN8M5S"
	webhookTimestamp := "1700000000"

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(webhookID))
	h.Write([]byte("."))
	h.Write([]byte(webhookTimestamp))
	h.Write([]byte("."))
	h.Write(body)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	headers := http.Header{}
	headers.Set("webhook-id", webhookID)
	headers.Set("webhook-timestamp", webhookTimestamp)
	headers.Set("webhook-signature", "v1,"+signature)

	return headers
}
