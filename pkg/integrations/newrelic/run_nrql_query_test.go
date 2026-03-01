package newrelic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunNRQLQuery__Setup(t *testing.T) {
	component := &RunNRQLQuery{}

	t.Run("missing query -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "",
			},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("timeout exceeds max -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query":   "SELECT count(*) FROM Transaction SINCE 1 hour ago",
				"timeout": 999,
			},
		})

		require.ErrorContains(t, err, "timeout cannot exceed 120 seconds")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "SELECT count(*) FROM Transaction SINCE 1 hour ago",
			},
		})

		require.NoError(t, err)
	})
}

func Test__RunNRQLQuery__Execute(t *testing.T) {
	component := &RunNRQLQuery{}

	t.Run("successful query -> emits results", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"actor": {
								"account": {
									"nrql": {
										"results": [{"count": 42567}]
									}
								}
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "SELECT count(*) FROM Transaction SINCE 1 hour ago",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "newrelic.nrqlResult", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "api.newrelic.com/graphql")
		assert.Equal(t, "test-user-api-key", req.Header.Get("Api-Key"))
	})

	t.Run("GraphQL error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": null,
						"errors": [{"message": "Invalid NRQL query"}]
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "INVALID QUERY",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "GraphQL error")
	})

	t.Run("empty results -> emits empty array", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"actor": {
								"account": {
									"nrql": {
										"results": []
									}
								}
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accountId":  "12345",
				"region":     "US",
				"userApiKey": "test-user-api-key",
				"licenseKey": "test-license-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "SELECT count(*) FROM Transaction WHERE error IS TRUE SINCE 1 hour ago",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
	})
}

func Test__RunNRQLQuery__OutputChannels(t *testing.T) {
	component := &RunNRQLQuery{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
