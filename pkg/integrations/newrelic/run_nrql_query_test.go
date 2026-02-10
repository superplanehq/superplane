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

	// Verify required fields
	var accountIDField, queryField *bool
	for _, field := range config {
		if field.Name == "accountId" {
			accountIDField = &field.Required
		}
		if field.Name == "query" {
			queryField = &field.Required
		}
	}

	require.NotNil(t, accountIDField)
	assert.True(t, *accountIDField)
	require.NotNil(t, queryField)
	assert.True(t, *queryField)
}

func TestClient_RunNRQLQuery_Success(t *testing.T) {
	t.Run("successful query -> returns results", func(t *testing.T) {
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction SINCE 1 hour ago", 10)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, response.Results, 1)
		assert.Equal(t, float64(1523), response.Results[0]["count"])
		require.NotNil(t, response.Metadata)
		assert.Equal(t, []string{"Transaction"}, response.Metadata.EventTypes)
		assert.Equal(t, int64(1707559740000), response.Metadata.TimeWindow.Begin)

		// Verify request
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "test-key", httpCtx.Requests[0].Header.Get("Api-Key"))
		assert.Equal(t, "application/json", httpCtx.Requests[0].Header.Get("Content-Type"))

		// Verify GraphQL query structure
		bodyBytes, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var gqlRequest GraphQLRequest
		err = json.Unmarshal(bodyBytes, &gqlRequest)
		require.NoError(t, err)
		assert.Contains(t, gqlRequest.Query, "account(id: 1234567)")
		assert.Contains(t, gqlRequest.Query, "timeout: 10")
		assert.Contains(t, gqlRequest.Query, "SELECT count(*) FROM Transaction SINCE 1 hour ago")
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT average(duration) FROM Transaction", 10)

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
							"results": []
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT * FROM NonExistent", 10)

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
							"results": [{"count": 100}]
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
			APIKey:       "eu-test-key",
			NerdGraphURL: "https://api.eu.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 7654321, "SELECT count(*) FROM Transaction", 10)

		require.NoError(t, err)
		require.NotNil(t, response)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.eu.newrelic.com/graphql", httpCtx.Requests[0].URL.String())
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT * FORM Transaction", 10)

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
			APIKey:       "invalid-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction", 10)

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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "INVALID QUERY", 10)

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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction", 10)

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to decode GraphQL response")
	})

	t.Run("missing actor in response -> returns error", func(t *testing.T) {
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
			APIKey:       "test-key",
			NerdGraphURL: "https://api.newrelic.com/graphql",
			http:         httpCtx,
		}

		response, err := client.RunNRQLQuery(context.Background(), 1234567, "SELECT count(*) FROM Transaction", 10)

		require.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "missing actor")
	})
}

func TestRunNRQLQuery_Setup(t *testing.T) {
	t.Run("valid configuration -> no error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "1234567",
				"query":     "SELECT count(*) FROM Transaction",
				"timeout":   10,
			},
		}

		err := component.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("missing accountId -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"query": "SELECT count(*) FROM Transaction",
			},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accountId is required")
	})

	t.Run("missing query -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "1234567",
			},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query is required")
	})

	t.Run("invalid timeout -> returns error", func(t *testing.T) {
		component := &RunNRQLQuery{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "1234567",
				"query":     "SELECT count(*) FROM Transaction",
				"timeout":   150, // exceeds max of 120
			},
		}

		err := component.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be between 0 and 120")
	})
}
