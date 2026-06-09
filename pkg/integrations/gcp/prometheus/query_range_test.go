package prometheus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__QueryRange__Setup(t *testing.T) {
	q := &QueryRange{}
	setup := func(cfg map[string]any) error {
		return q.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"query": "up", "lookbackPeriod": "6h"}))
	})

	t.Run("missing query -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"lookbackPeriod": "1h"}), "query is required")
	})

	t.Run("missing lookbackPeriod -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"query": "up"}), "lookbackPeriod is required")
	})

	t.Run("invalid lookbackPeriod -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"query": "up", "lookbackPeriod": "3y"}), "invalid lookbackPeriod")
	})
}

func Test__QueryRange__Execute(t *testing.T) {
	q := &QueryRange{}

	t.Run("range query derives the window from the lookback period", func(t *testing.T) {
		var gotURL string
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				gotURL = url
				return []byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"up"},"values":[[1767225600,"1"],[1767225660,"0"]]}]}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "up", "lookbackPeriod": "1h"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.prometheus.queryRange", state.Type)
		assert.Contains(t, gotURL, "/location/global/prometheus/api/v1/query_range?")
		assert.Contains(t, gotURL, "query=up")
		// 1h window uses a 60s step; start/end are derived (Unix seconds).
		assert.Contains(t, gotURL, "step=60s")
		assert.Contains(t, gotURL, "start=")
		assert.Contains(t, gotURL, "end=")

		payload := firstPayload(t, state)
		assert.Equal(t, "matrix", payload["resultType"])
		assert.Equal(t, 1, payload["seriesCount"])
		assert.Equal(t, "1h", payload["lookbackPeriod"])
		assert.Equal(t, "60s", payload["step"])
		assert.NotEmpty(t, payload["start"])
		assert.NotEmpty(t, payload["end"])
	})

	t.Run("invalid lookback fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "up", "lookbackPeriod": "99h"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "invalid lookbackPeriod")
	})
}
