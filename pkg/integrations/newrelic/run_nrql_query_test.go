package newrelic

import (
	"context"
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

func TestRunNRQLQuery_Name(t *testing.T) {
	component := &RunNRQLQuery{}
	assert.Equal(t, "newrelic.runNRQLQuery", component.Name())
}

func TestRunNRQLQuery_Label(t *testing.T) {
	component := &RunNRQLQuery{}
	assert.Equal(t, "Run NRQL Query", component.Label())
}

func TestRunNRQLQuery_Configuration(t *testing.T) {
	component := &RunNRQLQuery{}
	config := component.Configuration()

	assert.NotEmpty(t, config)
	assert.Len(t, config, 2) // account and query only, no timeout

	// Verify required fields
	var accountIDField, queryField *bool
	for _, field := range config {
		if field.Name == "account" {
			accountIDField = &field.Required
		}
		if field.Name == "query" {
			queryField = &field.Required
		}
	}

	require.NotNil(t, accountIDField)
	assert.True(t, *accountIDField) // account is now required
	require.NotNil(t, queryField)
	assert.True(t, *queryField)

	// Verify no timeout field
	for _, field := range config {
		assert.NotEqual(t, "timeout", field.Name, "timeout field should not be present in configuration")
	}
}

func TestRunNRQLQuery_Actions(t *testing.T) {
	component := &RunNRQLQuery{}
	actions := component.Actions()

	require.Len(t, actions, 1)
	assert.Equal(t, "poll", actions[0].Name)
	assert.False(t, actions[0].UserAccessible)
}

func TestClient_RunNRQLQuery_Success(t *testing.T) {
	t.Run("successful query -> returns results immediately", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [
								{
									"count": 1523
								}
							],
							"metadata": {
								"eventTypes": ["Transaction"],
								"messages": [],
								"timeWindow": {
									"begin": 1707559740000,
									"end": 1707563340000
								}
							},
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction SINCE 1 hour ago")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Results, 1)
		assert.Equal(t, float64(1523), response.Results[0]["count"])
		require.NotNil(t, response.Metadata)
		assert.Equal(t, []string{"Transaction"}, response.Metadata.EventTypes)
		assert.Equal(t, int64(1707559740000), response.Metadata.TimeWindow.Begin)

		// Verify queryProgress shows completed
		require.NotNil(t, response.QueryProgress)
		assert.True(t, response.QueryProgress.Completed)

		// Verify request
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "test-key", httpCtx.Requests[0].Header.Get("Api-Key"))
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))

		// Verify GraphQL query structure uses async: true
		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var gqlRequest GraphQLRequest
		err = json.Unmarshal(bodyBytes, &gqlRequest)
		require.NoError(t, err)
		assert.Contains(t, gqlRequest.Query, "account(id: 1234567)")
		assert.Contains(t, gqlRequest.Query, "timeout: 10")
		assert.Contains(t, gqlRequest.Query, "async: true")
		assert.Contains(t, gqlRequest.Query, "queryProgress")
		assert.Contains(t, gqlRequest.Query, "SELECT count(*) FROM Transaction SINCE 1 hour ago")
	})

	t.Run("async query not completed -> returns queryProgress", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": null,
							"queryProgress": {
								"queryId": "abc-123-def",
								"completed": false,
								"retryAfter": 10,
								"retryDeadline": 1707563340000,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT * FROM Transaction SINCE 24 hours ago")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.QueryProgress)
		assert.Equal(t, "abc-123-def", response.QueryProgress.QueryId)
		assert.False(t, response.QueryProgress.Completed)
		assert.Equal(t, 10, response.QueryProgress.RetryAfter)
	})

	t.Run("query with totalResult -> returns aggregated result", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [
								{
									"average": 123.45
								}
							],
							"totalResult": {
								"average": 123.45
							},
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT average(duration) FROM Transaction")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.TotalResult)
		assert.Equal(t, float64(123.45), response.TotalResult["average"])
	})

	t.Run("empty results -> returns empty array", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [],
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT * FROM NonExistent")

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Empty(t, response.Results)
	})

	t.Run("EU region -> uses EU endpoint", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [{"count": 100}],
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "eu-test-key",
			NerdGraphURL: "https://api.eu.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 7654321, "SELECT count(*) FROM Transaction")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.eu.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
	})
}

func TestClient_PollNRQLQuery(t *testing.T) {
	t.Run("poll completed -> returns results", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrqlQueryProgress": {
							"results": [{"count": 999}],
							"queryProgress": {
								"queryId": "abc-123-def",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.PollNRQLQuery(context.Background(), 1234567, "abc-123-def")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Results, 1)
		assert.Equal(t, float64(999), response.Results[0]["count"])
		require.NotNil(t, response.QueryProgress)
		assert.True(t, response.QueryProgress.Completed)

		// Verify GraphQL query uses nrqlQueryProgress
		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var gqlRequest GraphQLRequest
		err = json.Unmarshal(bodyBytes, &gqlRequest)
		require.NoError(t, err)
		assert.Contains(t, gqlRequest.Query, "nrqlQueryProgress")
		assert.Contains(t, gqlRequest.Query, "abc-123-def")
	})

	t.Run("poll still running -> returns queryProgress not completed", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrqlQueryProgress": {
							"results": null,
							"queryProgress": {
								"queryId": "abc-123-def",
								"completed": false,
								"retryAfter": 10,
								"retryDeadline": 1707563340000,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.PollNRQLQuery(context.Background(), 1234567, "abc-123-def")

		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.QueryProgress)
		assert.False(t, response.QueryProgress.Completed)
		assert.Equal(t, 10, response.QueryProgress.RetryAfter)
	})
}

func TestClient_RunNRQLQuery_Errors(t *testing.T) {
	t.Run("invalid NRQL syntax -> returns GraphQL error", func(t *testing.T) {
		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": null
					}
				}
			},
			"errors": [
				{
					"message": "NRQL Syntax Error: Error at line 1 position 8, unexpected 'FORM'",
					"path": ["actor", "account", "nrql"]
				}
			]
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT * FORM Transaction")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "GraphQL errors")
		assert.Contains(t, err.Error(), "NRQL Syntax Error")
	})

	t.Run("authentication error -> returns error", func(t *testing.T) {
		responseJSON := `{
			"error": {
				"title": "Unauthorized",
				"message": "Invalid API key"
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "invalid-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "Unauthorized")
	})

	t.Run("multiple GraphQL errors -> returns all errors", func(t *testing.T) {
		responseJSON := `{
			"errors": [
				{
					"message": "First error"
				},
				{
					"message": "Second error"
				}
			]
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "INVALID QUERY")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "First error")
		assert.Contains(t, err.Error(), "Second error")
	})

	t.Run("malformed response -> returns error", func(t *testing.T) {
		responseJSON := `{invalid json`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to decode GraphQL response")
	})

	t.Run("missing nrql data in response -> returns error", func(t *testing.T) {
		responseJSON := `{
			"data": {}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		client := &Client{
			UserAPIKey:   "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction")

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "missing nrql or nrqlQueryProgress")
	})
}

func TestRunNRQLQuery_Setup(t *testing.T) {
	t.Run("valid configuration -> no error", func(t *testing.T) {
		component := &RunNRQLQuery{}

		accountsJSON := `{
			"data": {
				"actor": {
					"accounts": [
						{"id": 1234567, "name": "Main Account"}
					]
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(accountsJSON)),
					Header:     make(http.Header),
				},
			},
		}

		ctx := core.SetupContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "SELECT count(*) FROM Transaction",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		assert.NoError(t, err)

		// Verify metadata was set
		metadata := ctx.Metadata.Get().(RunNRQLQueryNodeMetadata)
		assert.True(t, metadata.Manual)
		assert.NotNil(t, metadata.Account)
		assert.Equal(t, int64(1234567), metadata.Account.ID)
	})

	t.Run("missing account -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"query": "SELECT count(*) FROM Transaction",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account is required")
	})

	t.Run("missing query -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"account": "1234567",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query is required")
	})

	t.Run("account not found -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}

		accountsJSON := `{
			"data": {
				"actor": {
					"accounts": [
						{"id": 9999999, "name": "Other Account"}
					]
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(accountsJSON)),
					Header:     make(http.Header),
				},
			},
		}

		ctx := core.SetupContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "SELECT count(*) FROM Transaction",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account ID 1234567 not found")
	})
}

func TestRunNRQLQuery_Setup_TemplateGuard(t *testing.T) {
	component := &RunNRQLQuery{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"userApiKey": "test-key",
			"site":       "US",
		},
	}

	t.Run("unresolved account template -> error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"account": "{{account_id}}",
				"query":   "SELECT count(*) FROM Transaction",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		}
		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved template variable")
		assert.Contains(t, err.Error(), "{{account_id}}")
	})

	t.Run("unresolved query template -> error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "{{nrql_query}}",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		}
		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved template variable")
		assert.Contains(t, err.Error(), "{{nrql_query}}")
	})
}

func TestRunNRQLQuery_Execute_TemplateGuard(t *testing.T) {
	component := &RunNRQLQuery{}

	t.Run("unresolved account_id template -> error, no API call", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"account": "{{account_id}}",
				"query":   "SELECT count(*) FROM Transaction",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
		}
		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved template variable")
		assert.Contains(t, err.Error(), "{{account_id}}")
	})

	t.Run("unresolved query template -> error, no API call", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "{{nrql_query}}",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
		}
		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved template variable")
	})
}

func TestRunNRQLQuery_Execute_DataFallback(t *testing.T) {
	t.Run("account from ctx.Data fallback -> success", func(t *testing.T) {
		component := &RunNRQLQuery{}

		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [{"count": 42}],
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				// account is intentionally missing from config
				"query": "SELECT count(*) FROM Transaction",
			},
			Data: map[string]any{
				"accountId": "7654321",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			ExecutionState: executionState,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		payloadMap := executionState.Payloads[0].(map[string]any)
		payload := payloadMap["data"].(RunNRQLQueryPayload)
		assert.Equal(t, "7654321", payload.AccountID)
	})
}

func TestRunNRQLQuery_Execute(t *testing.T) {
	t.Run("string account ID, sync completion -> success", func(t *testing.T) {
		component := &RunNRQLQuery{}

		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": [{"count": 10}],
							"queryProgress": {
								"queryId": "",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 0
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "test-key",
				"site":       "US",
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "SELECT count(*) FROM Transaction",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)

		// Verify emission
		require.Len(t, executionState.Payloads, 1)
		payloadMap := executionState.Payloads[0].(map[string]any)
		payload := payloadMap["data"].(RunNRQLQueryPayload)
		assert.Equal(t, "1234567", payload.AccountID)
		assert.Equal(t, "SELECT count(*) FROM Transaction", payload.Query)
	})

	t.Run("async query not completed -> schedules poll", func(t *testing.T) {
		component := &RunNRQLQuery{}

		responseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrql": {
							"results": null,
							"queryProgress": {
								"queryId": "async-query-123",
								"completed": false,
								"retryAfter": 10,
								"retryDeadline": 1707563340000,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"userApiKey": "test-key",
				"site":       "US",
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"account": "1234567",
				"query":   "SELECT * FROM Transaction SINCE 24 hours ago",
			},
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)

		// Should NOT have emitted results
		assert.Empty(t, executionState.Payloads)
		assert.False(t, executionState.Finished)

		// Should have scheduled a poll action
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)

		// Should have stored metadata for polling
		metadata := metadataCtx.Get().(RunNRQLQueryExecutionMetadata)
		assert.Equal(t, "async-query-123", metadata.QueryId)
		assert.Equal(t, int64(1234567), metadata.AccountID)
		assert.Equal(t, "SELECT * FROM Transaction SINCE 24 hours ago", metadata.Query)
	})

	t.Run("invalid account ID string -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"account": "not-a-number",
				"query":   "SELECT count(*) FROM Transaction",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid account ID 'not-a-number'")
	})
}

func TestRunNRQLQuery_HandleAction_Poll(t *testing.T) {
	t.Run("poll completed -> emits results", func(t *testing.T) {
		component := &RunNRQLQuery{}

		pollResponseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrqlQueryProgress": {
							"results": [{"count": 500}],
							"queryProgress": {
								"queryId": "async-query-123",
								"completed": true,
								"retryAfter": 0,
								"retryDeadline": 0,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(pollResponseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"queryId":   "async-query-123",
				"accountId": float64(1234567),
				"query":     "SELECT * FROM Transaction SINCE 24 hours ago",
			},
		}

		ctx := core.ActionContext{
			Name: "poll",
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       &contexts.RequestContext{},
		}

		err := component.HandleAction(ctx)
		require.NoError(t, err)

		// Should have emitted results
		require.Len(t, executionState.Payloads, 1)
		payloadMap := executionState.Payloads[0].(map[string]any)
		payload := payloadMap["data"].(RunNRQLQueryPayload)
		assert.Equal(t, "1234567", payload.AccountID)
		assert.Equal(t, "SELECT * FROM Transaction SINCE 24 hours ago", payload.Query)
		assert.Equal(t, float64(500), payload.Results[0]["count"])
	})

	t.Run("poll still running -> reschedules", func(t *testing.T) {
		component := &RunNRQLQuery{}

		pollResponseJSON := `{
			"data": {
				"actor": {
					"account": {
						"nrqlQueryProgress": {
							"results": null,
							"queryProgress": {
								"queryId": "async-query-123",
								"completed": false,
								"retryAfter": 10,
								"retryDeadline": 1707563340000,
								"resultExpiration": 1707563940000
							}
						}
					}
				}
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(pollResponseJSON)),
					Header:     make(http.Header),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"queryId":   "async-query-123",
				"accountId": float64(1234567),
				"query":     "SELECT * FROM Transaction SINCE 24 hours ago",
			},
		}
		requestCtx := &contexts.RequestContext{}

		ctx := core.ActionContext{
			Name: "poll",
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"userApiKey": "test-key",
					"site":       "US",
				},
			},
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		}

		err := component.HandleAction(ctx)
		require.NoError(t, err)

		// Should NOT have emitted results
		assert.Empty(t, executionState.Payloads)
		assert.False(t, executionState.Finished)

		// Should have re-scheduled a poll
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, PollInterval, requestCtx.Duration)
	})

	t.Run("already finished -> noop", func(t *testing.T) {
		component := &RunNRQLQuery{}

		executionState := &contexts.ExecutionStateContext{
			Finished: true,
		}

		ctx := core.ActionContext{
			Name:           "poll",
			ExecutionState: executionState,
		}

		err := component.HandleAction(ctx)
		require.NoError(t, err)
	})

	t.Run("unknown action -> error", func(t *testing.T) {
		component := &RunNRQLQuery{}

		ctx := core.ActionContext{
			Name: "unknown",
		}

		err := component.HandleAction(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action: unknown")
	})
}
