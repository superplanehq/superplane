package render

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Render_OnDeploy__Setup(t *testing.T) {
	trigger := &OnDeploy{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "rnd_test"},
		Metadata: Metadata{
			Workspace: &WorkspaceMetadata{
				ID:   "usr-123",
				Plan: "professional",
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"cursor":"x","service":{"id":"srv-cukouhrtq21c73e9scng","name":"backend-api"}}]`,
				)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		HTTP:        httpCtx,
		Metadata:    metadataCtx,
		Integration: integrationCtx,
		Configuration: map[string]any{
			"service":    "srv-cukouhrtq21c73e9scng",
			"eventTypes": []string{"deploy_ended"},
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:   webhookStrategyIntegration,
		EventTypes: []string{"deploy_ended"},
	}, webhookConfiguration)

	nodeMetadata, ok := metadataCtx.Metadata.(OnResourceEventMetadata)
	require.True(t, ok)
	require.NotNil(t, nodeMetadata.Service)
	assert.Equal(t, "srv-cukouhrtq21c73e9scng", nodeMetadata.Service.ID)
	assert.Equal(t, "backend-api", nodeMetadata.Service.Name)
}

func Test__Render_OnDeploy__Setup__OrganizationWorkspace(t *testing.T) {
	trigger := &OnDeploy{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "rnd_test"},
		Metadata: Metadata{
			Workspace: &WorkspaceMetadata{
				ID:   "tea-123",
				Plan: "organization",
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"cursor":"x","service":{"id":"srv-cukouhrtq21c73e9scng","name":"backend-api"}}]`,
				)),
			},
		},
	}

	err := trigger.Setup(core.TriggerContext{
		HTTP:        httpCtx,
		Metadata:    metadataCtx,
		Integration: integrationCtx,
		Configuration: map[string]any{
			"service": "srv-cukouhrtq21c73e9scng",
		},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookConfiguration, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	require.True(t, ok)
	assert.Equal(t, WebhookConfiguration{
		Strategy:     webhookStrategyResourceType,
		ResourceType: webhookResourceTypeDeploy,
		EventTypes:   []string{"deploy_ended"},
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
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "missing signature headers")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("expired timestamp -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		expiredHeaders := buildSignedHeadersWithTimestamp(
			secret,
			body,
			strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10),
		)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       expiredHeaders,
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "timestamp expired")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		invalidHeaders := buildSignedHeaders(secret, body)
		invalidHeaders.Set("webhook-signature", "v1,invalid")

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       invalidHeaders,
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
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
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
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
			Configuration: map[string]any{"service": "srv-other"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
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
				"service":    "srv-cukouhrtq21c73e9scng",
				"eventTypes": []string{"deploy_started"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("default event filter -> event emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evj-cukouhrtq21c73e9scng","timestamp":"2026-02-05T16:00:01.000000Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deployId":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}
		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, map[string]any{
			"eventId":     "evj-cukouhrtq21c73e9scng",
			"deployId":    "dep-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "succeeded",
		}, eventCtx.Payloads[0].Data)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/v1/events/evj-cukouhrtq21c73e9scng")
	})

	t.Run("multiple signatures header -> accepts matching v1 signature", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		validHeaders := buildSignedHeaders(secret, body)
		validSignature := strings.TrimPrefix(validHeaders.Get("webhook-signature"), "v1,")
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evj-cukouhrtq21c73e9scng","timestamp":"2026-02-05T16:00:01.000000Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deployId":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}

		headersWithMultipleSignatures := http.Header{}
		headersWithMultipleSignatures.Set("webhook-id", validHeaders.Get("webhook-id"))
		headersWithMultipleSignatures.Set("webhook-timestamp", validHeaders.Get("webhook-timestamp"))
		headersWithMultipleSignatures.Set("webhook-signature", "v1,invalid v1,"+validSignature)

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headersWithMultipleSignatures,
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, map[string]any{
			"eventId":     "evj-cukouhrtq21c73e9scng",
			"deployId":    "dep-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "succeeded",
		}, eventCtx.Payloads[0].Data)
	})

	t.Run("whsec secret format -> accepts decoded signing key", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		rawSecret := "test-secret"
		webhookSecret := "whsec_" + base64.RawStdEncoding.EncodeToString([]byte(rawSecret))
		headers := buildSignedHeaders(rawSecret, body)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"evj-cukouhrtq21c73e9scng","timestamp":"2026-02-05T16:00:01.000000Z","serviceId":"srv-cukouhrtq21c73e9scng","type":"deploy_ended","details":{"deployId":"dep-cukouhrtq21c73e9scng","status":"live"}}`,
					)),
				},
			},
		}

		status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			HTTP:          httpCtx,
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
			Webhook:       &contexts.NodeWebhookContext{Secret: webhookSecret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
		assert.Equal(t, map[string]any{
			"eventId":     "evj-cukouhrtq21c73e9scng",
			"deployId":    "dep-cukouhrtq21c73e9scng",
			"serviceId":   "srv-cukouhrtq21c73e9scng",
			"serviceName": "backend-api",
			"status":      "succeeded",
		}, eventCtx.Payloads[0].Data)
	})
}

func Test__Render_OnDeploy__HandleWebhook__WithoutEventResolution(t *testing.T) {
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
	eventCtx := &contexts.EventContext{}

	status, webhookErr := trigger.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Configuration: map[string]any{"service": "srv-cukouhrtq21c73e9scng"},
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        eventCtx,
	})

	assert.Equal(t, http.StatusOK, status)
	require.NoError(t, webhookErr)
	require.Equal(t, 1, eventCtx.Count())
	assert.Equal(t, "render.deploy.ended", eventCtx.Payloads[0].Type)
	assert.Equal(t, map[string]any{
		"eventId":     "evj-cukouhrtq21c73e9scng",
		"serviceId":   "srv-cukouhrtq21c73e9scng",
		"serviceName": "backend-api",
		"status":      "succeeded",
	}, eventCtx.Payloads[0].Data)
}
