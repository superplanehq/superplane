package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Webhook__Setup(t *testing.T) {
	t.Run("sets metadata when missing", func(t *testing.T) {
		webhook := &Webhook{}
		metadataCtx := &contexts.MetadataContext{Metadata: Metadata{}}
		webhookCtx := &contexts.WebhookContext{}

		ctx := core.TriggerContext{
			Configuration: Configuration{Authentication: "signature"},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
		}

		require.NoError(t, webhook.Setup(ctx))

		metadata, ok := metadataCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotEmpty(t, metadata.URL)
		require.Equal(t, "signature", metadata.Authentication)
	})

	t.Run("keeps metadata when URL and auth match", func(t *testing.T) {
		webhook := &Webhook{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{
				URL:            "existing-url",
				Authentication: "signature",
			},
		}

		ctx := core.TriggerContext{
			Configuration: Configuration{Authentication: "signature"},
			Metadata:      metadataCtx,
			Webhook:       &contexts.WebhookContext{},
		}

		require.NoError(t, webhook.Setup(ctx))

		metadata, ok := metadataCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.Equal(t, "existing-url", metadata.URL)
		require.Equal(t, "signature", metadata.Authentication)
	})

	t.Run("updates auth when configuration changes", func(t *testing.T) {
		webhook := &Webhook{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{
				URL:            "existing-url",
				Authentication: "signature",
			},
		}

		ctx := core.TriggerContext{
			Configuration: Configuration{Authentication: "bearer"},
			Metadata:      metadataCtx,
			Webhook:       &contexts.WebhookContext{},
		}

		require.NoError(t, webhook.Setup(ctx))

		metadata, ok := metadataCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.Equal(t, "existing-url", metadata.URL)
		require.Equal(t, "bearer", metadata.Authentication)
	})
}

func Test__Webhook__HandleAction__ResetAuthentication(t *testing.T) {
	t.Run("resets signature authentication", func(t *testing.T) {
		webhook := &Webhook{}
		webhookCtx := &contexts.WebhookContext{Secret: "secret-key"}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{Authentication: "signature"},
		}

		result, err := webhook.HandleAction(core.TriggerActionContext{
			Name:          "resetAuthentication",
			Configuration: Configuration{Authentication: "signature"},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
		})

		require.NoError(t, err)
		require.Equal(t, "secret-key", result["secret"])
	})

	t.Run("resets bearer authentication", func(t *testing.T) {
		webhook := &Webhook{}
		webhookCtx := &contexts.WebhookContext{Secret: "bearer-secret"}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{Authentication: "bearer"},
		}

		result, err := webhook.HandleAction(core.TriggerActionContext{
			Name:          "resetAuthentication",
			Configuration: Configuration{Authentication: "bearer"},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
		})

		require.NoError(t, err)
		require.Equal(t, "bearer-secret", result["secret"])
	})

	t.Run("resets header token authentication", func(t *testing.T) {
		webhook := &Webhook{}
		webhookCtx := &contexts.WebhookContext{Secret: "header-token-secret"}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{Authentication: "header_token"},
		}

		result, err := webhook.HandleAction(core.TriggerActionContext{
			Name:          "resetAuthentication",
			Configuration: Configuration{Authentication: "header_token"},
			Metadata:      metadataCtx,
			Webhook:       webhookCtx,
		})

		require.NoError(t, err)
		require.Equal(t, "header-token-secret", result["secret"])
	})

	t.Run("rejects unsupported authentication", func(t *testing.T) {
		webhook := &Webhook{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: Metadata{Authentication: "none"},
		}

		_, err := webhook.HandleAction(core.TriggerActionContext{
			Name:          "resetAuthentication",
			Configuration: Configuration{Authentication: "none"},
			Metadata:      metadataCtx,
			Webhook:       &contexts.WebhookContext{},
		})

		require.Error(t, err)
	})
}

func Test__Webhook__HandleWebhook(t *testing.T) {
	t.Run("rejects payloads larger than MaxEventSize", func(t *testing.T) {
		webhook := &Webhook{}
		body := make([]byte, MaxEventSize+1)
		ctx, _ := webhookRequestContext(body, "none", "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusRequestEntityTooLarge, status)
		require.Error(t, err)
	})

	t.Run("rejects invalid JSON payloads", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte("not-json"), "none", "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusBadRequest, status)
		require.Error(t, err)
	})

	t.Run("rejects missing signature header", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "signature", "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusForbidden, status)
		require.Error(t, err)
	})

	t.Run("rejects invalid signature format", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "signature", "secret")
		ctx.Headers.Set("X-Signature-256", "sha256=")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusForbidden, status)
		require.Error(t, err)
	})

	t.Run("rejects invalid signature", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "signature", "secret")
		ctx.Headers.Set("X-Signature-256", "sha256=invalid")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusForbidden, status)
		require.Error(t, err)
	})

	t.Run("accepts valid signature and emits event", func(t *testing.T) {
		webhook := &Webhook{}
		body := []byte(`{"foo":"bar"}`)
		ctx, eventCtx := webhookRequestContext(body, "signature", "secret")
		signature := computeSignature("secret", body)
		ctx.Headers.Set("X-Signature-256", "sha256="+signature)

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusOK, status)
		require.NoError(t, err)

		require.Equal(t, 1, eventCtx.Count())
		payload := eventCtx.Payloads[0]
		require.Equal(t, "webhook", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)
		require.Contains(t, data, "body")
		require.Contains(t, data, "headers")
	})

	t.Run("rejects missing bearer token", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "bearer", "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("rejects invalid bearer token", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "bearer", "secret")
		ctx.Headers.Set("Authorization", "Bearer wrong")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("accepts bearer token and masks header", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, eventCtx := webhookRequestContext([]byte(`{"ok":true}`), "bearer", "secret")
		ctx.Headers.Set("Authorization", "Bearer secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusOK, status)
		require.NoError(t, err)

		require.Equal(t, 1, eventCtx.Count())
		payload := eventCtx.Payloads[0]
		require.Equal(t, "webhook", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)

		headers, ok := data["headers"].(http.Header)
		require.True(t, ok)
		require.Equal(t, "Bearer ********", headers.Get("Authorization"))
	})

	t.Run("rejects missing header token", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "header_token", "secret")
		config, ok := ctx.Configuration.(map[string]any)
		require.True(t, ok)
		config["headerName"] = "X-Test-Token"

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("rejects invalid header token", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, _ := webhookRequestContext([]byte(`{"ok":true}`), "header_token", "secret")
		config, ok := ctx.Configuration.(map[string]any)
		require.True(t, ok)
		config["headerName"] = "X-Test-Token"
		ctx.Headers.Set("X-Test-Token", "wrong")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Error(t, err)
	})

	t.Run("accepts header token and masks configured header", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, eventCtx := webhookRequestContext([]byte(`{"ok":true}`), "header_token", "secret")
		config, ok := ctx.Configuration.(map[string]any)
		require.True(t, ok)
		config["headerName"] = "X-Test-Token"
		ctx.Headers.Set("X-Test-Token", "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusOK, status)
		require.NoError(t, err)

		require.Equal(t, 1, eventCtx.Count())
		payload := eventCtx.Payloads[0]
		require.Equal(t, "webhook", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)

		headers, ok := data["headers"].(http.Header)
		require.True(t, ok)
		require.Equal(t, "********", headers.Get("X-Test-Token"))
	})

	t.Run("accepts header token with default header name", func(t *testing.T) {
		webhook := &Webhook{}
		ctx, eventCtx := webhookRequestContext([]byte(`{"ok":true}`), "header_token", "secret")
		ctx.Headers.Set(DefaultHeaderTokenName, "secret")

		status, err := webhook.HandleWebhook(ctx)
		require.Equal(t, http.StatusOK, status)
		require.NoError(t, err)

		require.Equal(t, 1, eventCtx.Count())
		payload := eventCtx.Payloads[0]
		require.Equal(t, "webhook", payload.Type)

		data, ok := payload.Data.(map[string]any)
		require.True(t, ok)

		headers, ok := data["headers"].(http.Header)
		require.True(t, ok)
		require.Equal(t, "********", headers.Get(DefaultHeaderTokenName))
	})
}

func webhookRequestContext(body []byte, authentication string, secret string) (core.WebhookRequestContext, *contexts.EventContext) {
	eventCtx := &contexts.EventContext{}
	webhookCtx := &contexts.WebhookContext{Secret: secret}

	return core.WebhookRequestContext{
		Body:          body,
		Headers:       http.Header{},
		Configuration: map[string]any{"authentication": authentication},
		Webhook:       webhookCtx,
		Events:        eventCtx,
	}, eventCtx
}

func computeSignature(key string, data []byte) string {
	hash := hmac.New(sha256.New, []byte(key))
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}
