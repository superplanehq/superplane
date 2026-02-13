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
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Slack__Sync(t *testing.T) {
	s := &Slack{}

	t.Run("metadata already set -> no prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{URL: "https://slack.example.com"},
		}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.NoError(t, err)
		assert.Nil(t, integrationCtx.BrowserAction)
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

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"botToken":      "token-123",
				"signingSecret": "secret-123",
			},
			BrowserAction: &core.BrowserAction{URL: "https://example.com"},
		}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		assert.Nil(t, integrationCtx.BrowserAction)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "https://workspace.slack.com", metadata.URL)
		assert.Equal(t, "T123", metadata.TeamID)
		assert.Equal(t, "team", metadata.Team)
		assert.Equal(t, "U123", metadata.UserID)
		assert.Equal(t, "bot", metadata.User)
		assert.Equal(t, "B123", metadata.BotID)
	})

	t.Run("no tokens -> browser action includes manifest", func(t *testing.T) {
		integrationID := uuid.NewString()
		integrationCtx := &contexts.IntegrationContext{IntegrationID: integrationID}

		err := s.Sync(core.SyncContext{
			Integration: integrationCtx,
			BaseURL:     "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		require.NotEmpty(t, integrationCtx.BrowserAction.URL)

		manifestURL, err := url.Parse(integrationCtx.BrowserAction.URL)
		require.NoError(t, err)
		manifestParam := manifestURL.Query().Get("manifest_json")
		require.NotEmpty(t, manifestParam)

		var manifest map[string]any
		require.NoError(t, json.Unmarshal([]byte(manifestParam), &manifest))

		settings := manifest["settings"].(map[string]any)
		eventSubs := settings["event_subscriptions"].(map[string]any)
		interactivity := settings["interactivity"].(map[string]any)

		assert.Equal(t, fmt.Sprintf("https://app.example.com/api/v1/integrations/%s/events", integrationID), eventSubs["request_url"])
		assert.Equal(t, fmt.Sprintf("https://app.example.com/api/v1/integrations/%s/interactions", integrationID), interactivity["request_url"])
	})

	t.Run("PUBLIC_API_BASE_PATH set to /api/v1 -> no duplication in URLs", func(t *testing.T) {
		// Save original env var and restore after test
		originalPath := os.Getenv("PUBLIC_API_BASE_PATH")
		t.Cleanup(func() {
			if originalPath == "" {
				os.Unsetenv("PUBLIC_API_BASE_PATH")
			} else {
				os.Setenv("PUBLIC_API_BASE_PATH", originalPath)
			}
		})

		os.Setenv("PUBLIC_API_BASE_PATH", "/api/v1")

		integrationID := uuid.NewString()
		integrationCtx := &contexts.IntegrationContext{IntegrationID: integrationID}

		err := s.Sync(core.SyncContext{
			Integration: integrationCtx,
			BaseURL:     "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)

		manifestURL, err := url.Parse(integrationCtx.BrowserAction.URL)
		require.NoError(t, err)
		manifestParam := manifestURL.Query().Get("manifest_json")
		require.NotEmpty(t, manifestParam)

		var manifest map[string]any
		require.NoError(t, json.Unmarshal([]byte(manifestParam), &manifest))

		settings := manifest["settings"].(map[string]any)
		eventSubs := settings["event_subscriptions"].(map[string]any)
		interactivity := settings["interactivity"].(map[string]any)

		// Should not have /api/v1/api/v1 duplication
		eventURL := eventSubs["request_url"].(string)
		interactivityURL := interactivity["request_url"].(string)

		assert.Equal(t, fmt.Sprintf("https://app.example.com/api/v1/integrations/%s/events", integrationID), eventURL)
		assert.Equal(t, fmt.Sprintf("https://app.example.com/api/v1/integrations/%s/interactions", integrationID), interactivityURL)
		assert.NotContains(t, eventURL, "/api/v1/api/v1", "URL should not contain /api/v1 duplication")
		assert.NotContains(t, interactivityURL, "/api/v1/api/v1", "URL should not contain /api/v1 duplication")
	})

	t.Run("PUBLIC_API_BASE_PATH set to custom path -> path included in URLs", func(t *testing.T) {
		// Save original env var and restore after test
		originalPath := os.Getenv("PUBLIC_API_BASE_PATH")
		t.Cleanup(func() {
			if originalPath == "" {
				os.Unsetenv("PUBLIC_API_BASE_PATH")
			} else {
				os.Setenv("PUBLIC_API_BASE_PATH", originalPath)
			}
		})

		os.Setenv("PUBLIC_API_BASE_PATH", "/custom")

		integrationID := uuid.NewString()
		integrationCtx := &contexts.IntegrationContext{IntegrationID: integrationID}

		err := s.Sync(core.SyncContext{
			Integration: integrationCtx,
			BaseURL:     "https://app.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)

		manifestURL, err := url.Parse(integrationCtx.BrowserAction.URL)
		require.NoError(t, err)
		manifestParam := manifestURL.Query().Get("manifest_json")
		require.NotEmpty(t, manifestParam)

		var manifest map[string]any
		require.NoError(t, json.Unmarshal([]byte(manifestParam), &manifest))

		settings := manifest["settings"].(map[string]any)
		eventSubs := settings["event_subscriptions"].(map[string]any)
		interactivity := settings["interactivity"].(map[string]any)

		// Should include /custom path
		assert.Equal(t, fmt.Sprintf("https://app.example.com/custom/api/v1/integrations/%s/events", integrationID), eventSubs["request_url"])
		assert.Equal(t, fmt.Sprintf("https://app.example.com/custom/api/v1/integrations/%s/interactions", integrationID), interactivity["request_url"])
	})
}

func Test__Slack__ReadAndVerify(t *testing.T) {
	s := &Slack{}

	t.Run("missing timestamp header -> error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "https://example.com", bytes.NewBufferString("payload"))
		req.Header.Set("X-Slack-Signature", "v0=signature")

		_, err := s.readAndVerify(core.HTTPRequestContext{
			Request: req,
			Integration: &contexts.IntegrationContext{
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
			Integration: &contexts.IntegrationContext{
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
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"signingSecret": secret},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, body, got)
	})
}

func Test__Slack__HandleEvent__Challenge(t *testing.T) {
	s := &Slack{}

	t.Run("missing challenge -> 400", func(t *testing.T) {
		payload := EventPayload{
			Type: "url_verification",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		s.handleEvent(core.HTTPRequestContext{
			Logger:   logrus.NewEntry(logrus.New()),
			Response: recorder,
		}, body)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("challenge present -> 200 with body", func(t *testing.T) {
		payload := EventPayload{
			Type:      "url_verification",
			Challenge: "challenge-token",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		s.handleEvent(core.HTTPRequestContext{
			Logger:   logrus.NewEntry(logrus.New()),
			Response: recorder,
		}, body)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "challenge-token", recorder.Body.String())
	})
}
