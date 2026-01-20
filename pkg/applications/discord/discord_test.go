package discord

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Discord__Sync(t *testing.T) {
	d := &Discord{}

	t.Run("missing webhook URL -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "webhookUrl is required")
	})

	t.Run("invalid webhook URL format -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"webhookUrl": "https://example.com/webhook",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"webhookUrl": "https://example.com/webhook"},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid webhook URL format")
	})

	t.Run("valid webhook URL -> verifies and sets ready", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://discord.com/api/webhooks/123456789/abc-def-token", req.URL.String())
			assert.Equal(t, http.MethodGet, req.Method)
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"type": 1,
				"guild_id": "987654321",
				"channel_id": "111222333",
				"name": "Test Webhook"
			}`), nil
		})

		webhookURL := "https://discord.com/api/webhooks/123456789/abc-def-token"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"webhookUrl": webhookURL,
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"webhookUrl": webhookURL},
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Contains(t, appCtx.StateDescription, "111222333")

		metadata, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "123456789", metadata.WebhookID)
	})

	t.Run("webhook verification fails -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusNotFound, `{"message": "Unknown Webhook"}`), nil
		})

		webhookURL := "https://discord.com/api/webhooks/123456789/invalid-token"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"webhookUrl": webhookURL,
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"webhookUrl": webhookURL},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify webhook")
	})

	t.Run("discordapp.com URL is accepted", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"type": 1,
				"channel_id": "111222333",
				"name": "Test Webhook"
			}`), nil
		})

		webhookURL := "https://discordapp.com/api/webhooks/123456789/abc-def-token"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"webhookUrl": webhookURL,
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"webhookUrl": webhookURL},
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})
}
