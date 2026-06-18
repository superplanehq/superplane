package cloudsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateDatabase__Setup(t *testing.T) {
	c := &CreateDatabase{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing instance -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "app_db"}), "instance is required")
	})

	t.Run("missing name -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"instance": "my-instance"}), "name is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"instance": "my-instance", "name": "app_db"}))
	})
}

func Test__CreateDatabase__Execute(t *testing.T) {
	c := &CreateDatabase{}

	t.Run("creates the database and emits its details", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				postBody, _ = body.(map[string]any)
				return []byte(doneOperation), nil
			},
			getFunc: func(ctx context.Context, url string) ([]byte, error) {
				return []byte(`{"name":"app_db","instance":"my-instance","project":"my-project","charset":"UTF8","collation":"en_US.UTF8","selfLink":"https://x/app_db"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance", "name": "app_db"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.database", state.Type)
		assert.Contains(t, postURL, "/projects/my-project/instances/my-instance/databases")
		assert.Equal(t, "app_db", postBody["name"])

		data := firstData(t, state)
		assert.Equal(t, "app_db", data["name"])
		assert.Equal(t, "my-instance", data["instance"])
		assert.Equal(t, "UTF8", data["charset"])
	})

	t.Run("surfaces a failed operation", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				return []byte(`{"name":"op-1","status":"DONE","error":{"errors":[{"message":"database already exists"}]}}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance", "name": "app_db"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "database already exists")
	})

	t.Run("missing instance fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "app_db"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "instance is required")
	})
}
