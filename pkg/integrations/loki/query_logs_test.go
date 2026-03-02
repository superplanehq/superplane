package loki

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

func Test__QueryLogs__Setup(t *testing.T) {
	component := &QueryLogs{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing query -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "",
			},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": `{job="superplane"}`,
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with all fields -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": `{job="superplane"}`,
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"limit": "50",
			},
		})

		require.NoError(t, err)
	})
}

func Test__QueryLogs__Execute(t *testing.T) {
	component := &QueryLogs{}

	t.Run("successful query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "success",
						"data": {
							"resultType": "streams",
							"result": [
								{
									"stream": {"job": "superplane"},
									"values": [
										["1708000000000000000", "log line 1"],
										["1708000015000000000", "log line 2"]
									]
								}
							]
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": `{job="superplane"}`,
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"limit": "100",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "loki.queryLogs", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Contains(t, req.URL.String(), "/loki/api/v1/query_range")
	})

	t.Run("query failure -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("parse error")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": `{invalid`,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query logs")
	})

	t.Run("query without optional fields", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "success",
						"data": {
							"resultType": "streams",
							"result": []
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": `{job="superplane"}`,
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
	})
}

func Test__QueryLogs__OutputChannels(t *testing.T) {
	component := &QueryLogs{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
