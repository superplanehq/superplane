package teams

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Teams__Sync(t *testing.T) {
	s := &Teams{}

	t.Run("metadata already set and installed -> ready", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{AppID: "test-app-id", Installed: true},
			Configuration: map[string]any{
				"appId":       "test-app-id",
				"appPassword": "test-password",
			},
		}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})

	t.Run("no credentials -> browser action", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.URL, "portal.azure.com")
	})

	t.Run("valid credentials -> pending with manifest download", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "oauth2/v2.0/token")
			return jsonResponse(http.StatusOK, `{
				"access_token": "test-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`), nil
		})

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"appId":       "test-app-id",
				"appPassword": "test-password",
			},
		}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.NoError(t, err)

		// Not ready yet — user hasn't installed the Teams app (installed=false)
		assert.NotEqual(t, "ready", integrationCtx.State)

		// A BrowserAction with the manifest ZIP download should be set
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.URL, "data:application/zip;base64,")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Finish Azure Setup")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "webhook URL")

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "test-app-id", metadata.AppID)
		assert.False(t, metadata.Installed)
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, `{"error": "invalid_client"}`), nil
		})

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"appId":       "bad-app-id",
				"appPassword": "bad-password",
			},
		}

		err := s.Sync(core.SyncContext{Integration: integrationCtx})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify credentials")
	})
}

func Test__Teams__HandleRequest__MissingAuth(t *testing.T) {
	s := &Teams{}

	body := []byte(`{"type": "message", "text": "hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/messages", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	s.HandleRequest(core.HTTPRequestContext{
		Logger:   logrus.NewEntry(logrus.New()),
		Request:  req,
		Response: recorder,
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{AppID: "test-app-id"},
		},
	})

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func Test__Teams__HandleRequest__InvalidAuthFormat(t *testing.T) {
	s := &Teams{}

	body := []byte(`{"type": "message", "text": "hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/messages", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Basic dGVzdDp0ZXN0")
	recorder := httptest.NewRecorder()

	s.HandleRequest(core.HTTPRequestContext{
		Logger:   logrus.NewEntry(logrus.New()),
		Request:  req,
		Response: recorder,
		Integration: &contexts.IntegrationContext{
			Metadata: Metadata{AppID: "test-app-id"},
		},
	})

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func Test__Teams__HasBotMention(t *testing.T) {
	t.Run("bot mentioned -> true", func(t *testing.T) {
		activity := Activity{
			Recipient: ChannelAccount{ID: "28:bot-id"},
			Entities: []Entity{
				{
					Type:      "mention",
					Mentioned: &ChannelAccount{ID: "28:bot-id"},
				},
			},
		}

		assert.True(t, hasBotMention(activity))
	})

	t.Run("other user mentioned -> false", func(t *testing.T) {
		activity := Activity{
			Recipient: ChannelAccount{ID: "28:bot-id"},
			Entities: []Entity{
				{
					Type:      "mention",
					Mentioned: &ChannelAccount{ID: "29:other-user"},
				},
			},
		}

		assert.False(t, hasBotMention(activity))
	})

	t.Run("no mentions -> false", func(t *testing.T) {
		activity := Activity{
			Recipient: ChannelAccount{ID: "28:bot-id"},
			Entities:  []Entity{},
		}

		assert.False(t, hasBotMention(activity))
	})
}

func Test__Teams__SubscriptionApplies(t *testing.T) {
	t.Run("mention subscription matches mention event", func(t *testing.T) {
		config := SubscriptionConfiguration{EventTypes: []string{"mention"}}
		configMap := map[string]any{}
		b, _ := json.Marshal(config)
		_ = json.Unmarshal(b, &configMap)

		assert.True(t, containsEventType(configMap, "mention"))
		assert.False(t, containsEventType(configMap, "message"))
	})

	t.Run("message subscription matches message event", func(t *testing.T) {
		config := SubscriptionConfiguration{EventTypes: []string{"message"}}
		configMap := map[string]any{}
		b, _ := json.Marshal(config)
		_ = json.Unmarshal(b, &configMap)

		assert.True(t, containsEventType(configMap, "message"))
		assert.False(t, containsEventType(configMap, "mention"))
	})
}

// containsEventType is a test helper to check subscription config.
func containsEventType(configMap map[string]any, eventType string) bool {
	c := SubscriptionConfiguration{}
	if err := mapstructure.Decode(configMap, &c); err != nil {
		return false
	}

	for _, t := range c.EventTypes {
		if t == eventType {
			return true
		}
	}

	return false
}
