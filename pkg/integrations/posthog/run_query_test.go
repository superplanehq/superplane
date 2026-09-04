package posthog

import (
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

func Test__RunQuery__Setup(t *testing.T) {
	action := &RunQuery{}

	t.Run("valid configuration", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectId": "12345",
				"query":     "SELECT event FROM events LIMIT 1",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing project returns error", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{"query": "SELECT event FROM events LIMIT 1"},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("blank query returns error in HogQL mode", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{"projectId": "12345", "mode": QueryModeHogQL, "query": "   "},
		})

		require.ErrorContains(t, err, "query is required")
	})

	t.Run("builder mode needs no query", func(t *testing.T) {
		err := action.Setup(core.SetupContext{
			Configuration: map[string]any{"projectId": "12345", "mode": QueryModeBuilder},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration format returns decode error", func(t *testing.T) {
		err := action.Setup(core.SetupContext{Configuration: "invalid-config"})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__RunQuery__Execute(t *testing.T) {
	action := &RunQuery{}

	queryResponse := `{"results":[["signup","user_1"],["purchase","user_2"]],"columns":["event","distinct_id"],"types":["String","String"]}`

	t.Run("runs the query and emits named rows", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(queryResponse)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := action.Execute(core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"projectId": "12345",
				"query":     "SELECT event, distinct_id FROM events LIMIT 2",
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "phx_test"}},
			ExecutionState: executionState,
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://us.posthog.com/api/projects/12345/query/", httpContext.Requests[0].URL.String())

		assert.True(t, executionState.Passed)
		require.Len(t, executionState.Payloads, 1)

		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "posthog.queryResult", payload["type"])

		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "12345", data["projectId"])
		assert.Equal(t, 2, data["rowCount"])
		assert.Equal(t, []string{"event", "distinct_id"}, data["columns"])

		rows, ok := data["rows"].([]any)
		require.True(t, ok)
		require.Len(t, rows, 2)
		assert.Equal(t, map[string]any{"event": "signup", "distinct_id": "user_1"}, rows[0])
	})

	t.Run("missing project returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := action.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"query": "SELECT event FROM events"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "phx_test"}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "project is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing query returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := action.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"projectId": "12345", "mode": QueryModeHogQL},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "phx_test"}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "query is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("PostHog error is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"detail":"Syntax error near FROM"}`)),
				},
			},
		}

		err := action.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"projectId": "12345", "query": "SELECT bad FROM"},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "phx_test"}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to run query")
	})
}
