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

func Test__GetDatabase__Setup(t *testing.T) {
	g := &GetDatabase{}
	setup := func(cfg map[string]any) error {
		return g.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing instance -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"database": "app_db"}), "instance is required")
	})

	t.Run("missing database -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"instance": "my-instance"}), "database is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"instance": "my-instance", "database": "app_db"}))
	})
}

func Test__GetDatabase__Execute(t *testing.T) {
	g := &GetDatabase{}

	t.Run("fetches the database and emits its details", func(t *testing.T) {
		var getURL string
		mc := &mockClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				getURL = url
				return []byte(`{"name":"app_db","instance":"my-instance","project":"my-project","charset":"UTF8","collation":"en_US.UTF8","selfLink":"https://x/app_db"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := g.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance", "database": "app_db"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.database", state.Type)
		assert.Contains(t, getURL, "/projects/my-project/instances/my-instance/databases/app_db")

		data := firstData(t, state)
		assert.Equal(t, "app_db", data["name"])
		assert.Equal(t, "en_US.UTF8", data["collation"])
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
			Configuration:  map[string]any{"instance": "my-instance", "database": "missing"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get database")
	})
}
