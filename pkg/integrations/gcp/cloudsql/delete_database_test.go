package cloudsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteDatabase__Setup(t *testing.T) {
	d := &DeleteDatabase{}
	setup := func(cfg map[string]any) error {
		return d.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
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

func Test__DeleteDatabase__Execute(t *testing.T) {
	d := &DeleteDatabase{}

	t.Run("deletes the database and emits a confirmation", func(t *testing.T) {
		var deleteURL string
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte(doneOperation), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance", "database": "app_db"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.database", state.Type)
		assert.Contains(t, deleteURL, "/projects/my-project/instances/my-instance/databases/app_db")

		data := firstData(t, state)
		assert.Equal(t, "app_db", data["name"])
		assert.Equal(t, "my-instance", data["instance"])
		assert.Equal(t, true, data["deleted"])
	})

	t.Run("surfaces a failed delete operation", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				return []byte(`{"name":"op-2","status":"DONE","error":{"errors":[{"message":"database is in use"}]}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance", "database": "app_db"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "database is in use")
	})
}
