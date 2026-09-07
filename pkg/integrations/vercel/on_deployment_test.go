package vercel

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func jsonResponseBody(value string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(value))
}

func signBody(secret string, body []byte) http.Header {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	headers := http.Header{}
	headers.Set("x-vercel-signature", hex.EncodeToString(mac.Sum(nil)))
	return headers
}

func webhookBody(eventType string) []byte {
	body := map[string]any{
		"type": eventType,
		"payload": map[string]any{
			"target": "production",
			"deployment": map[string]any{
				"id":         "dpl_1",
				"name":       "my-app",
				"url":        "my-app.vercel.app",
				"readyState": "READY",
				"projectId":  "prj_1",
			},
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return encoded
}

func Test__Vercel_OnDeployment__Setup(t *testing.T) {
	t.Run("requests shared account-level webhook", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"accessToken": "vercel_token_123"},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       jsonResponseBody(`{"id":"prj_1","name":"my-app"}`),
				},
			},
		}

		err := (&OnDeployment{}).Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Metadata:      &contexts.MetadataContext{},
			Integration:   integrationCtx,
			Configuration: map[string]any{"project": "prj_1"},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		_, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.True(t, ok)
	})

	t.Run("unknown project -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       jsonResponseBody(`{"error":{"code":"NOT_FOUND"}}`),
				},
			},
		}

		err := (&OnDeployment{}).Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Metadata:      &contexts.MetadataContext{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			Configuration: map[string]any{"project": "prj_missing"},
		})

		require.ErrorContains(t, err, "failed to fetch Vercel project prj_missing")
	})
}

func Test__Vercel_OnDeployment__HandleWebhook(t *testing.T) {
	const secret = "signing-secret-123"

	t.Run("missing signature -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          webhookBody("deployment.succeeded"),
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "missing signature header")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		headers := signBody("wrong-secret", webhookBody("deployment.succeeded"))

		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          webhookBody("deployment.succeeded"),
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusForbidden, status)
		assert.ErrorContains(t, webhookErr, "invalid signature")
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("unsupported event type -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		body := webhookBody("deployment.check-rerequested")

		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       signBody(secret, body),
			Configuration: map[string]any{},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("project filter mismatch -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		body := webhookBody("deployment.succeeded")

		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       signBody(secret, body),
			Configuration: map[string]any{"project": "prj_other"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("unselected event type -> ignored", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		body := webhookBody("deployment.created")

		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       signBody(secret, body),
			Configuration: map[string]any{"eventTypes": []string{"deployment.succeeded"}},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		assert.Zero(t, eventCtx.Count())
	})

	t.Run("matching project and event -> emitted", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		body := webhookBody("deployment.succeeded")

		status, _, webhookErr := (&OnDeployment{}).HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       signBody(secret, body),
			Configuration: map[string]any{"project": "prj_1"},
			Webhook:       &contexts.NodeWebhookContext{Secret: secret},
			Events:        eventCtx,
		})

		assert.Equal(t, http.StatusOK, status)
		require.NoError(t, webhookErr)
		require.Equal(t, 1, eventCtx.Count())

		emitted := eventCtx.Payloads[0]
		assert.Equal(t, "vercel.deployment.succeeded", emitted.Type)

		data := readMap(emitted.Data)
		assert.Equal(t, "deployment.succeeded", data["eventType"])
		assert.Equal(t, "dpl_1", data["deploymentId"])
		assert.Equal(t, "prj_1", data["projectId"])
		assert.Equal(t, "READY", data["readyState"])

		rawPayload := readMap(data["payload"])
		assert.NotEmpty(t, rawPayload, "raw Vercel payload is preserved")
	})
}
