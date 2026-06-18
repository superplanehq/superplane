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

	valid := map[string]any{"query": "up", "start": "2026-01-01T00:00:00Z", "end": "2026-01-02T00:00:00Z", "step": "60s"}

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(valid))
	})

	t.Run("missing query -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"start": "1", "end": "2", "step": "60s"}), "query is required")
	})

	t.Run("missing start -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"query": "up", "end": "2", "step": "60s"}), "start is required")
	})

	t.Run("missing end -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"query": "up", "start": "1", "step": "60s"}), "end is required")
	})

	t.Run("missing step -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"query": "up", "start": "1", "end": "2"}), "step is required")
	})
}

func Test__QueryRange__Execute(t *testing.T) {
	q := &QueryRange{}

	t.Run("range query passes the start/end/step through", func(t *testing.T) {
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
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "60s",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.prometheus.queryRange", state.Type)
		assert.Contains(t, gotURL, "/location/global/prometheus/api/v1/query_range?")
		assert.Contains(t, gotURL, "query=up")
		assert.Contains(t, gotURL, "step=60s")
		// start/end are passed through as provided (URL-encoded).
		assert.Contains(t, gotURL, "start=2026-01-01T00%3A00%3A00Z")
		assert.Contains(t, gotURL, "end=2026-01-02T00%3A00%3A00Z")

		payload := firstPayload(t, state)
		assert.Equal(t, "matrix", payload["resultType"])
		assert.Equal(t, 1, payload["seriesCount"])
		assert.Equal(t, "2026-01-01T00:00:00Z", payload["start"])
		assert.Equal(t, "2026-01-02T00:00:00Z", payload["end"])
		assert.Equal(t, "60s", payload["step"])
	})

	t.Run("missing step fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := q.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"query": "up", "start": "1", "end": "2"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "step is required")
	})
}
