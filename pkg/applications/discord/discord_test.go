package discord

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Discord__Sync(t *testing.T) {
	d := &Discord{}

	t.Run("missing bot token -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "botToken is required")
	})

	t.Run("empty bot token -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"botToken": ""},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "botToken is required")
	})

	t.Run("valid bot token -> verifies and sets ready", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://discord.com/api/v10/users/@me", req.URL.String())
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "Bot test-bot-token", req.Header.Get("Authorization"))
			return jsonResponse(http.StatusOK, `{
				"id": "123456789",
				"username": "TestBot",
				"discriminator": "0000",
				"bot": true
			}`), nil
		})

		botToken := "test-bot-token"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": botToken,
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"botToken": botToken},
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Contains(t, appCtx.StateDescription, "TestBot")

		metadata, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "123456789", metadata.BotID)
		assert.Equal(t, "TestBot", metadata.Username)
	})

	t.Run("bot token verification fails -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{"message": "401: Unauthorized"}`), nil
		})

		botToken := "invalid-token"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": botToken,
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   map[string]any{"botToken": botToken},
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify bot token")
	})
}

func Test__Discord__ListResources(t *testing.T) {
	d := &Discord{}

	t.Run("lists channels from guilds", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v10/users/@me/guilds" {
				return jsonResponse(http.StatusOK, `[
					{"id": "guild1", "name": "Test Server"}
				]`), nil
			}
			if req.URL.Path == "/api/v10/guilds/guild1/channels" {
				return jsonResponse(http.StatusOK, `[
					{"id": "channel1", "name": "general", "type": 0},
					{"id": "channel2", "name": "voice", "type": 2}
				]`), nil
			}
			return jsonResponse(http.StatusNotFound, `{}`), nil
		})

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "test-token",
			},
		}

		resources, err := d.ListResources("channel", core.ListResourcesContext{
			AppInstallation: appCtx,
			Logger:          logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		// Should only include text channel (type 0)
		require.Len(t, resources, 1)
		assert.Equal(t, "channel1", resources[0].ID)
		assert.Equal(t, "#general (Test Server)", resources[0].Name)
		assert.Equal(t, "channel", resources[0].Type)
	})

	t.Run("unknown resource type returns empty", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken": "test-token",
			},
		}

		resources, err := d.ListResources("unknown", core.ListResourcesContext{
			AppInstallation: appCtx,
			Logger:          logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
