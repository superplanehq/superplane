package newrelic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

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

func Test__NewRelic__Sync(t *testing.T) {
	n := &NewRelic{}

	t.Run("missing API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{},
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("empty API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"apiKey": "", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          &contexts.HTTPContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("invalid API key -> error", func(t *testing.T) {
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
				"apiKey": "NRAK-invalid-key",
				"site":   "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"apiKey": "NRAK-invalid-key", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate API key")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "NRAK-invalid-key", httpCtx.Requests[0].Header.Get("Api-Key"))
	})

	t.Run("valid API key -> sets ready", func(t *testing.T) {
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

		apiKey := "NRAK-test-api-key"
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": apiKey,
				"site":   "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"apiKey": apiKey, "site": "US"},
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

	t.Run("network error -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "NRAK-test-key",
				"site":   "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"apiKey": "NRAK-test-key", "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to validate API key")
	})
	t.Run("license key -> skips validation and sets ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// No requests expected for License Key
			},
		}

		apiKey := "license-key-1234567890" // Does not start with NRAK-
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": apiKey,
				"site":   "US",
			},
		}

		err := n.Sync(core.SyncContext{
			Configuration: map[string]any{"apiKey": apiKey, "site": "US"},
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		// Should have NO requests because validation and account fetching are skipped
		require.Len(t, httpCtx.Requests, 0)
		
		// Metadata should be set but empty accounts
		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Empty(t, metadata.Accounts)
	})
}

func Test__NewRelic__ListResources(t *testing.T) {
	n := &NewRelic{}

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
	t.Run("missing API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": "US",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("empty API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("valid API key -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "US",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "test-key", client.APIKey)
		assert.Equal(t, "https://api.newrelic.com/graphql", client.NerdGraphURL)
	})

	t.Run("valid EU API key -> success", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
				"site":   "EU",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "test-key", client.APIKey)
		assert.Equal(t, "https://api.eu.newrelic.com/graphql", client.NerdGraphURL)
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "test-key", httpCtx.Requests[0].Header.Get("Api-Key"))
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
			APIKey:       "license-key",
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
			APIKey:       strings.Repeat("x", 40),
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
			APIKey:       "invalid-key",
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		err := client.ValidateAPIKey(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "GraphQL errors")
		assert.Contains(t, err.Error(), "Invalid API key")
	})
}

func Test__NewRelic__Name(t *testing.T) {
	integration := &NewRelic{}
	assert.Equal(t, "newrelic", integration.Name())
}

func Test__NewRelic__Label(t *testing.T) {
	integration := &NewRelic{}
	assert.Equal(t, "New Relic", integration.Label())
}

func Test__NewRelic__Configuration(t *testing.T) {
	integration := &NewRelic{}
	config := integration.Configuration()
	assert.NotEmpty(t, config)
	assert.Equal(t, "site", config[0].Name)
	assert.True(t, config[0].Required)
	assert.Equal(t, "apiKey", config[1].Name)
	assert.True(t, config[1].Required)
	assert.True(t, config[1].Sensitive)
}
