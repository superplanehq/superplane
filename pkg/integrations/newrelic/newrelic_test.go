package newrelic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func Test__Newrelic__Sync(t *testing.T) {
	n := &Newrelic{}

	t.Run("no keys provided -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"site": "US"},
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one API key is required")
	})

	t.Run("empty keys -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "",
				"licenseKey": "",
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"userApiKey": "", "licenseKey": "", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one API key is required")
	})

	t.Run("invalid user API key -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusUnauthorized, `{
					"error": {
						"title": "Unauthorized",
						"message": "Invalid API key"
					}
				}`),
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-invalid-key",
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"userApiKey": "NRAK-invalid-key", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate User API Key")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "NRAK-invalid-key", httpCtx.Requests[0].Header.Get("Api-Key"))
	})

	t.Run("valid user API key -> validates and sets ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": {
							"user": {
								"name": "Test User",
								"email": "test@example.com"
							}
						}
					}
				}`),
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": {
							"accounts": [
								{"id": 123456, "name": "Test Account"}
							]
						}
					}
				}`),
			},
		}

		userAPIKey := "NRAK-test-api-key"
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": userAPIKey,
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"userApiKey": userAPIKey, "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "NRAK-test-api-key", httpCtx.Requests[0].Header.Get("Api-Key"))
	})

	t.Run("network error with user API key -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-key",
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"userApiKey": "NRAK-test-key", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate User API Key")
	})

	t.Run("only license key -> skips validation and sets ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"licenseKey": "license-key-1234567890",
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"licenseKey": "license-key-1234567890", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 0)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Empty(t, metadata.Accounts)
	})

	t.Run("both keys -> validates user API key and sets ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": {
							"user": {
								"name": "Test User",
								"email": "test@example.com"
							}
						}
					}
				}`),
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": {
							"accounts": [
								{"id": 123456, "name": "Test Account"}
							]
						}
					}
				}`),
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-api-key",
				"licenseKey": "license-key-12345",
				"site":       "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-api-key",
				"licenseKey": "license-key-12345",
				"site":       "US",
			},
			Integration: integrationCtx,
			HTTP:        httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
	})
}

func Test__Newrelic__ListResources(t *testing.T) {
	n := &Newrelic{}

	t.Run("lists accounts from metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Accounts: []Account{
					{ID: 123456, Name: "Test Account"},
					{ID: 789012, Name: "Production Account"},
				},
			},
		}

		resources, err := n.ListResources("account", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "account", resources[0].Type)
		assert.Equal(t, "123456", resources[0].ID)
		assert.Equal(t, "Test Account", resources[0].Name)
		assert.Equal(t, "789012", resources[1].ID)
		assert.Equal(t, "Production Account", resources[1].Name)
	})

	t.Run("unknown resource type returns empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Accounts: []Account{}},
		}

		resources, err := n.ListResources("unknown", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("empty metadata returns empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Accounts: []Account{}},
		}

		resources, err := n.ListResources("account", core.ListResourcesContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func Test__Client__NewClient(t *testing.T) {
	t.Run("no keys -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": "US",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one API key is required")
	})

	t.Run("empty keys -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "",
				"licenseKey": "",
				"site":       "US",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one API key is required")
	})

	t.Run("user API key only -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-key",
				"site":       "US",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "NRAK-test-key", client.UserAPIKey)
		assert.Equal(t, "", client.LicenseKey)
		assert.Equal(t, "https://api.newrelic.com/graphql", client.NerdGraphURL)
		assert.Equal(t, "https://metric-api.newrelic.com/metric/v1", client.MetricBaseURL)
	})

	t.Run("license key only -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"licenseKey": "license-key-12345",
				"site":       "US",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "", client.UserAPIKey)
		assert.Equal(t, "license-key-12345", client.LicenseKey)
		assert.Equal(t, "https://api.newrelic.com/graphql", client.NerdGraphURL)
	})

	t.Run("both keys -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-key",
				"licenseKey": "license-key-12345",
				"site":       "US",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "NRAK-test-key", client.UserAPIKey)
		assert.Equal(t, "license-key-12345", client.LicenseKey)
	})

	t.Run("EU region -> correct URLs", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-test-key",
				"site":       "EU",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://api.eu.newrelic.com/graphql", client.NerdGraphURL)
		assert.Equal(t, "https://metric-api.eu.newrelic.com/metric/v1", client.MetricBaseURL)
	})
}

func Test__Client__ValidateAPIKey(t *testing.T) {
	t.Run("successful request -> validates successfully", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": {
							"user": {
								"name": "Test User",
								"email": "test@example.com"
							}
						}
					}
				}`),
			},
		}

		client := &Client{
			UserAPIKey:   "NRAK-test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "NRAK-test-key", httpCtx.Requests[0].Header.Get("Api-Key"))
	})

	t.Run("successful request with missing actor -> validates successfully", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusOK, `{
					"data": {
						"actor": null
					}
				}`),
			},
		}

		client := &Client{
			UserAPIKey:   "NRAK-test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.NoError(t, err)
	})

	t.Run("long garbage key -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusUnauthorized, `{
					"error": {
						"title": "Unauthorized",
						"message": "Invalid API key"
					}
				}`),
			},
		}

		client := &Client{
			UserAPIKey:   strings.Repeat("x", 40),
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unauthorized")
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusUnauthorized, `{
					"error": {
						"title": "Unauthorized",
						"message": "Invalid API key"
					}
				}`),
			},
		}

		client := &Client{
			UserAPIKey:   "invalid-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unauthorized")
	})

	t.Run("GraphQL errors -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(http.StatusOK, `{
					"errors": [
						{"message": "Invalid API key"}
					]
				}`),
			},
		}

		client := &Client{
			UserAPIKey:   "NRAK-test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "GraphQL errors")
		assert.Contains(t, err.Error(), "Invalid API key")
	})
}

func Test__NewrelicWebhookHandler__Setup(t *testing.T) {
	handler := &NewrelicWebhookHandler{}

	t.Run("Setup - auto-provisions destination and channel", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				nerdGraphSearchDestinationsResponse(nil), // Not found
				nerdGraphDestinationResponse("dest-123"),
				nerdGraphSearchChannelsResponse(nil), // Not found
				nerdGraphChannelResponse("chan-456"),
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "NRAK-TEST",
				"site":       "US",
			},
		}

		ctx := core.WebhookHandlerContext{
			Integration: integrationCtx,
			Webhook:     &mockWebhook{id: uuid.New(), url: "https://superplane.io/webhook/123", config: map[string]any{"account": "123456"}, secret: "test-secret"},
			HTTP:        httpCtx,
		}

		result, err := handler.Setup(ctx)
		require.NoError(t, err)

		metadata, ok := result.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "dest-123", metadata.DestinationID)
		assert.Equal(t, "chan-456", metadata.ChannelID)

		// Verify NerdGraph calls
		require.Len(t, httpCtx.Requests, 4)
	})
}

type mockWebhook struct {
	id     uuid.UUID
	url    string
	config any
	secret string
	meta   any
}

func (w *mockWebhook) GetID() string                 { return w.id.String() }
func (w *mockWebhook) GetURL() string                { return w.url }
func (w *mockWebhook) GetConfiguration() any         { return w.config }
func (w *mockWebhook) GetSecret() ([]byte, error)    { return []byte(w.secret), nil }
func (w *mockWebhook) SetSecret(secret []byte) error { w.secret = string(secret); return nil }
func (w *mockWebhook) GetMetadata() any              { return w.meta }
func (w *mockWebhook) SetMetadata(meta any)          { w.meta = meta }

func nerdGraphDestinationResponse(destID string) *http.Response {
	body := `{
		"data": {
			"aiNotificationsCreateDestination": {
				"destination": { "id": "` + destID + `" },
				"errors": []
			}
		}
	}`
	return jsonResponse(http.StatusOK, body)
}

func nerdGraphChannelResponse(channelID string) *http.Response {
	body := `{
		"data": {
			"aiNotificationsCreateChannel": {
				"channel": { "id": "` + channelID + `" },
				"errors": []
			}
		}
	}`
	return jsonResponse(http.StatusOK, body)
}

func nerdGraphSearchDestinationsResponse(destinations []map[string]string) *http.Response {
	var entities []string
	for _, d := range destinations {
		entities = append(entities, fmt.Sprintf(`{"id": "%s", "name": "%s"}`, d["id"], d["name"]))
	}
	body := `{
		"data": {
			"actor": {
				"account": {
					"aiNotifications": {
						"destinations": {
							"entities": [` + strings.Join(entities, ",") + `]
						}
					}
				}
			}
		}
	}`
	return jsonResponse(http.StatusOK, body)
}

func nerdGraphSearchChannelsResponse(channels []map[string]string) *http.Response {
	var entities []string
	for _, c := range channels {
		entities = append(entities, fmt.Sprintf(`{"id": "%s", "name": "%s"}`, c["id"], c["name"]))
	}
	body := `{
		"data": {
			"actor": {
				"account": {
					"aiNotifications": {
						"channels": {
							"entities": [` + strings.Join(entities, ",") + `]
						}
					}
				}
			}
		}
	}`
	return jsonResponse(http.StatusOK, body)
}

func Test__Newrelic__Name(t *testing.T) {
	integration := &Newrelic{}
	assert.Equal(t, "newrelic", integration.Name())
}

func Test__Newrelic__Label(t *testing.T) {
	integration := &Newrelic{}
	assert.Equal(t, "Newrelic", integration.Label())
}

func Test__Newrelic__Configuration(t *testing.T) {
	integration := &Newrelic{}
	config := integration.Configuration()
	assert.NotEmpty(t, config)
	assert.Len(t, config, 3)
	assert.Equal(t, "site", config[0].Name)
	assert.True(t, config[0].Required)
	assert.Equal(t, "userApiKey", config[1].Name)
	assert.False(t, config[1].Required)
	assert.True(t, config[1].Sensitive)
	assert.Equal(t, "licenseKey", config[2].Name)
	assert.False(t, config[2].Required)
	assert.True(t, config[2].Sensitive)
}
