package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Telegram__Sync(t *testing.T) {
	t.Run("new webhook secret -> registers and stores secret", func(t *testing.T) {
		var webhookPayload map[string]string
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/getMe"):
				return jsonResponse(http.StatusOK, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"testbot"}}`), nil
			case strings.HasSuffix(req.URL.Path, "/setWebhook"):
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &webhookPayload))
				return jsonResponse(http.StatusOK, `{"ok":true,"result":true}`), nil
			default:
				t.Fatalf("unexpected Telegram API request: %s", req.URL.Path)
				return nil, nil
			}
		})

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-token"},
		}

		err := (&Telegram{}).Sync(core.SyncContext{
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		secret := webhookPayload["secret_token"]
		require.NotEmpty(t, secret)
		assert.Regexp(t, `^[A-Za-z0-9_-]+$`, secret)
		require.Contains(t, integrationCtx.CurrentSecrets, "webhookSecret")
		assert.Equal(t, secret, string(integrationCtx.CurrentSecrets["webhookSecret"].Value))
	})

	t.Run("existing webhook secret -> reuses secret", func(t *testing.T) {
		var webhookPayload map[string]string
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/getMe"):
				return jsonResponse(http.StatusOK, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"testbot"}}`), nil
			case strings.HasSuffix(req.URL.Path, "/setWebhook"):
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &webhookPayload))
				return jsonResponse(http.StatusOK, `{"ok":true,"result":true}`), nil
			default:
				t.Fatalf("unexpected Telegram API request: %s", req.URL.Path)
				return nil, nil
			}
		})

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-token"},
		}
		require.NoError(t, integrationCtx.SetSecret("webhookSecret", []byte("existing-secret")))

		err := (&Telegram{}).Sync(core.SyncContext{
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "existing-secret", webhookPayload["secret_token"])
		assert.Equal(t, "existing-secret", string(integrationCtx.CurrentSecrets["webhookSecret"].Value))
	})

	t.Run("webhook registration failure -> does not store secret", func(t *testing.T) {
		withDefaultTransport(t, func(req *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(req.URL.Path, "/getMe"):
				return jsonResponse(http.StatusOK, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"testbot"}}`), nil
			case strings.HasSuffix(req.URL.Path, "/setWebhook"):
				return jsonResponse(http.StatusOK, `{"ok":false,"description":"Bad Request"}`), nil
			default:
				t.Fatalf("unexpected Telegram API request: %s", req.URL.Path)
				return nil, nil
			}
		})

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"botToken": "test-token"},
		}

		err := (&Telegram{}).Sync(core.SyncContext{
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.ErrorContains(t, err, "failed to set webhook")
		assert.NotContains(t, integrationCtx.CurrentSecrets, "webhookSecret")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})
}

type trackedReader struct {
	io.Reader
	read bool
}

func (r *trackedReader) Read(p []byte) (int, error) {
	r.read = true
	return r.Reader.Read(p)
}

func Test__Telegram__HandleRequest(t *testing.T) {
	const (
		secret        = "expected-secret"
		mentionUpdate = `{"update_id":1,"message":{"message_id":42,"chat":{"id":123,"type":"group"},"text":"@testbot hello","entities":[{"type":"mention","offset":0,"length":8}],"date":1737028800}}`
		forgedUpdate  = `{"update_id":1,"callback_query":{"id":"query-1","from":{"id":7,"is_bot":false,"first_name":"Attacker"},"message":{"message_id":42,"chat":{"id":123,"type":"group"},"date":1737028800},"data":"approve"}}`
	)

	tests := []struct {
		name         string
		storedSecret string
		header       string
		body         string
		wantStatus   int
		wantBodyRead bool
	}{
		{
			name:         "no stored secret -> processes legacy request",
			body:         mentionUpdate,
			wantStatus:   http.StatusOK,
			wantBodyRead: true,
		},
		{
			name:         "matching secret -> processes request",
			storedSecret: secret,
			header:       secret,
			body:         mentionUpdate,
			wantStatus:   http.StatusOK,
			wantBodyRead: true,
		},
		{
			name:         "missing secret header -> rejects before reading body",
			storedSecret: secret,
			body:         forgedUpdate,
			wantStatus:   http.StatusForbidden,
			wantBodyRead: false,
		},
		{
			name:         "wrong secret header -> rejects before reading body",
			storedSecret: secret,
			header:       "wrong-secret",
			body:         forgedUpdate,
			wantStatus:   http.StatusForbidden,
			wantBodyRead: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackedReader{Reader: strings.NewReader(test.body)}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/events", body)
			if test.header != "" {
				request.Header.Set("X-Telegram-Bot-Api-Secret-Token", test.header)
			}
			response := httptest.NewRecorder()
			integrationCtx := &contexts.IntegrationContext{
				Configuration: map[string]any{"botToken": "test-token"},
				Metadata: map[string]any{
					"username": "testbot",
					"botId":    float64(1),
				},
				Subscriptions: []contexts.Subscription{
					{Configuration: SubscriptionConfiguration{EventTypes: []string{"message_mention"}}},
					{Configuration: map[string]any{
						"type":         "button_click",
						"message_id":   float64(42),
						"chat_id":      "123",
						"execution_id": "b4fef823-3921-4270-8b1d-8c324bc624b2",
					}},
				},
			}
			if test.storedSecret != "" {
				require.NoError(t, integrationCtx.SetSecret("webhookSecret", []byte(test.storedSecret)))
			}

			(&Telegram{}).HandleRequest(core.HTTPRequestContext{
				Logger:      logrus.NewEntry(logrus.New()),
				Request:     request,
				Response:    response,
				Integration: integrationCtx,
			})

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantBodyRead, body.read)
		})
	}
}
