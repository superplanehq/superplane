package slack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Slack__Sync(t *testing.T) {
	s := &Slack{}

	t.Run("metadata already set -> no prompt", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{URL: "https://slack.example.com"},
		}

		err := s.Sync(core.SyncContext{AppInstallation: appCtx})

		require.NoError(t, err)
		assert.Nil(t, appCtx.BrowserAction)
	})

	t.Run("tokens configured -> auth ok -> ready", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "https://slack.com/api/auth.test", req.URL.String())
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "Bearer token-123", req.Header.Get("Authorization"))
			return jsonResponse(http.StatusOK, `{
				"ok": true,
				"url": "https://workspace.slack.com",
				"team": "team",
				"team_id": "T123",
				"user": "bot",
				"user_id": "U123",
				"bot_id": "B123"
			}`), nil
		})

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"botToken":      "token-123",
				"signingSecret": "secret-123",
			},
			BrowserAction: &core.BrowserAction{URL: "https://example.com"},
		}

		err := s.Sync(core.SyncContext{AppInstallation: appCtx})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Nil(t, appCtx.BrowserAction)

		metadata, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "https://workspace.slack.com", metadata.URL)
		assert.Equal(t, "T123", metadata.TeamID)
		assert.Equal(t, "team", metadata.Team)
		assert.Equal(t, "U123", metadata.UserID)
		assert.Equal(t, "bot", metadata.User)
		assert.Equal(t, "B123", metadata.BotID)
	})

	t.Run("no tokens -> browser action includes manifest", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}

		err := s.Sync(core.SyncContext{
			AppInstallation: appCtx,
			BaseURL:         "https://app.example.com",
			InstallationID:  "install-123",
		})

		require.NoError(t, err)
		require.NotNil(t, appCtx.BrowserAction)
		require.NotEmpty(t, appCtx.BrowserAction.URL)

		manifestURL, err := url.Parse(appCtx.BrowserAction.URL)
		require.NoError(t, err)
		manifestParam := manifestURL.Query().Get("manifest_json")
		require.NotEmpty(t, manifestParam)

		var manifest map[string]any
		require.NoError(t, json.Unmarshal([]byte(manifestParam), &manifest))

		settings := manifest["settings"].(map[string]any)
		eventSubs := settings["event_subscriptions"].(map[string]any)
		interactivity := settings["interactivity"].(map[string]any)

		assert.Equal(t, "https://app.example.com/api/v1/apps/install-123/events", eventSubs["request_url"])
		assert.Equal(t, "https://app.example.com/api/v1/apps/install-123/interactions", interactivity["request_url"])
	})
}

func Test__Slack__ReadAndVerify(t *testing.T) {
	s := &Slack{}

	t.Run("missing timestamp header -> error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com", bytes.NewBufferString("payload"))
		req.Header.Set("X-Slack-Signature", "v0=signature")

		_, err := s.readAndVerify(core.HTTPRequestContext{
			Request: req,
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"signingSecret": "secret"},
			},
		})

		require.ErrorContains(t, err, "missing X-Slack-Request-Timestamp")
	})

	t.Run("invalid signature -> error", func(t *testing.T) {
		body := []byte("payload")
		req := httptest.NewRequest(http.MethodPost, "https://example.com", bytes.NewBuffer(body))
		req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
		req.Header.Set("X-Slack-Signature", "v0=invalid")

		_, err := s.readAndVerify(core.HTTPRequestContext{
			Request: req,
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"signingSecret": "secret"},
			},
		})

		require.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature -> returns body", func(t *testing.T) {
		body := []byte("payload")
		timestamp := time.Now().Unix()
		secret := "secret"
		sigBase := fmt.Sprintf("v0:%d:%s", timestamp, string(body))
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(sigBase))
		signature := "v0=" + hex.EncodeToString(h.Sum(nil))

		req := httptest.NewRequest(http.MethodPost, "https://example.com", bytes.NewBuffer(body))
		req.Header.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", timestamp))
		req.Header.Set("X-Slack-Signature", signature)

		got, err := s.readAndVerify(core.HTTPRequestContext{
			Request: req,
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{"signingSecret": secret},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, body, got)
	})
}
