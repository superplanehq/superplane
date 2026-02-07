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
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Render_OnDeploy__Setup(t *testing.T) {
	trigger := &OnDeploy{}
	integrationCtx := &contexts.IntegrationContext{}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		Configuration: map[string]any{
			"serviceId":  "srv-cukouhrtq21c73e9scng",
			"eventTypes": []string{"deploy_ended"},
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:   renderWebhookStrategyIntegration,
		EventTypes: []string{"deploy_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnBuild__Setup(t *testing.T) {
	trigger := &OnBuild{}
	integrationCtx := &contexts.IntegrationContext{}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		Configuration: map[string]any{
			"serviceId":  "srv-cukouhrtq21c73e9scng",
			"eventTypes": []string{"build_ended"},
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:   renderWebhookStrategyIntegration,
		EventTypes: []string{"build_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnDeploy__Setup__OrganizationWorkspace(t *testing.T) {
	trigger := &OnDeploy{}
	integrationCtx := &contexts.IntegrationContext{
		Metadata: Metadata{
			OwnerID:       "tea-123",
			WorkspacePlan: "organization",
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		Configuration: map[string]any{
			"serviceId": "srv-cukouhrtq21c73e9scng",
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:     renderWebhookStrategyResourceType,
		ResourceType: renderWebhookResourceTypeDeploy,
		EventTypes:   []string{"deploy_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnBuild__Setup__OrganizationWorkspace(t *testing.T) {
	trigger := &OnBuild{}
	integrationCtx := &contexts.IntegrationContext{
		Metadata: Metadata{
			OwnerID:       "tea-123",
			WorkspacePlan: "organization",
		},
	}

	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
		Configuration: map[string]any{
			"serviceId": "srv-cukouhrtq21c73e9scng",
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:     renderWebhookStrategyResourceType,
		ResourceType: renderWebhookResourceTypeBuild,
		EventTypes:   []string{"build_ended"},
	}, webhookConfiguration)
}

func Test__Render_OnDeploy__HandleWebhook(t *testing.T) {
	trigger := &OnDeploy{}

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
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
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
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "invalid signature")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("unsupported event type -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}

		buildPayload := map[string]any{
			"type": "build_ended",
			"data": payload["data"],
		}
		buildBody, marshalErr := json.Marshal(buildPayload)
		require.NoError(t, marshalErr)
		buildHeaders := buildSignedHeaders(secret, buildBody)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          buildBody,
			Headers:       buildHeaders,
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("service filter mismatch -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"serviceId": "srv-other"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("event type filter does not match -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"serviceId":  "srv-cukouhrtq21c73e9scng",
				"eventTypes": []string{"deploy_started"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("default event filter -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload["data"], eventCtx.Payloads[0].Data)
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
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload["data"], eventCtx.Payloads[0].Data)
	})

	t.Run("whsec secret format -> accepts decoded signing key", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		rawSecret := "test-secret"
		webhookSecret := "whsec_" + base64.RawStdEncoding.EncodeToString([]byte(rawSecret))
		headers := buildSignedHeaders(rawSecret, body)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.WebhookContext{Secret: webhookSecret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, payload["data"], eventCtx.Payloads[0].Data)
	})
}

func Test__Render_OnBuild__HandleWebhook(t *testing.T) {
	trigger := &OnBuild{}

	payload := map[string]any{
		"type":      "build_ended",
		"timestamp": "2026-02-05T16:00:01.000000Z",
		"data": map[string]any{
			"id":          "evj-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "failed",
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	secret := "whsec-test"
	headers := buildSignedHeaders(secret, body)
	eventCtx := &contexts.EventContext{}

	status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Configuration: map[string]any{"serviceId": "srv-cukouhrtq21c73e9scng"},
		Webhook:       &contexts.WebhookContext{Secret: secret},
		Events:        eventCtx,
	})

	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, webhookErr)
	require.Equal(t, 1, eventCtx.Count())
	assert.Equal(t, "render.build.ended", eventCtx.Payloads[0].Type)
	assert.Equal(t, payload["data"], eventCtx.Payloads[0].Data)
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
