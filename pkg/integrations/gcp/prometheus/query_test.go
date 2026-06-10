package prometheus

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Query__Setup(t *testing.T) {
	q := &Query{}
	setup := func(cfg map[string]any) error {
		return q.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing query -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "query is required")
	})

	t.Run("valid query -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"query": "up"}))
	})
}

func Test__Query__Execute(t *testing.T) {
	q := &Query{}

	t.Run("instant query emits the parsed vector result", func(t *testing.T) {
		var gotURL string
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				gotURL = url
				return []byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up"},"value":[1767225600,"1"]}]}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "up"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.prometheus.query", state.Type)
		// Hits the GMP Prometheus-compatible frontend with the escaped query,
		// evaluated at "now" (no explicit time parameter).
		assert.Contains(t, gotURL, "/v1/projects/my-project/location/global/prometheus/api/v1/query?")
		assert.Contains(t, gotURL, "query=up")
		assert.NotContains(t, gotURL, "time=")

		payload := firstPayload(t, state)
		assert.Equal(t, "vector", payload["resultType"])
		assert.Equal(t, 1, payload["seriesCount"])
		require.Len(t, payload["result"].([]any), 1)
	})

	t.Run("scalar result reports a single value, not its array length", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				// A scalar result is a [timestamp, value] pair, not a list of series.
				return []byte(`{"status":"success","data":{"resultType":"scalar","result":[1767225600,"42"]}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "scalar(up)"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		payload := firstPayload(t, state)
		assert.Equal(t, "scalar", payload["resultType"])
		assert.Equal(t, 1, payload["seriesCount"])
	})

	t.Run("prometheus error status fails the execution", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return []byte(`{"status":"error","errorType":"bad_data","error":"parse error at char 3"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "up{"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "bad_data")
		assert.Contains(t, state.FailureMessage, "parse error")
	})

	t.Run("missing query fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "   "},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, strings.ToLower(state.FailureMessage), "query is required")
	})
}
