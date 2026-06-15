package cloudsql

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetInstance__Setup(t *testing.T) {
	g := &GetInstance{}
	setup := func(cfg map[string]any) error {
		return g.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing instance -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "instance is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"instance": "my-instance"}))
	})
}

func Test__GetInstance__Execute(t *testing.T) {
	g := &GetInstance{}

	t.Run("fetches the instance and emits its details", func(t *testing.T) {
		var getURL string
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				getURL = url
				return []byte(`{"name":"my-instance","state":"RUNNABLE","databaseVersion":"POSTGRES_16","region":"us-central1","connectionName":"my-project:us-central1:my-instance","selfLink":"https://x/my-instance","settings":{"tier":"db-f1-micro","dataDiskSizeGb":"10","edition":"ENTERPRISE"},"ipAddresses":[{"type":"PRIMARY","ipAddress":"34.41.10.20"}]}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.instance", state.Type)
		assert.Contains(t, getURL, "/projects/my-project/instances/my-instance")

		data := firstData(t, state)
		assert.Equal(t, "RUNNABLE", data["state"])
		assert.Equal(t, "db-f1-micro", data["tier"])
		assert.Equal(t, "34.41.10.20", data["ipAddress"])
		assert.Equal(t, "my-project:us-central1:my-instance", data["connectionName"])
	})

	t.Run("surfaces an API error", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return nil, fmt.Errorf("not found")
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "missing"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get instance")
	})
}
