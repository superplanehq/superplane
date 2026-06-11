package cloudsql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateInstance__Setup(t *testing.T) {
	c := &CreateInstance{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing name -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"}), "name is required")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "i1", "databaseVersion": "POSTGRES_16", "tier": "db-f1-micro"}), "region is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"name": "i1", "databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"}))
	})
}

func Test__CreateInstance__Execute(t *testing.T) {
	c := &CreateInstance{}

	t.Run("provisions the instance and emits the operation", func(t *testing.T) {
		var postURL string
		var postBody map[string]any
		mc := &mockClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, url string, body any) ([]byte, error) {
				postURL = url
				postBody, _ = body.(map[string]any)
				return []byte(`{"name":"op-123","status":"PENDING","targetId":"my-instance"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "my-instance", "databaseVersion": "POSTGRES_16",
				"region": "us-central1", "tier": "db-f1-micro", "diskSizeGb": 10, "edition": "ENTERPRISE",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.instance", state.Type)
		assert.Contains(t, postURL, "/projects/my-project/instances")
		assert.Equal(t, "my-instance", postBody["name"])
		assert.Equal(t, "POSTGRES_16", postBody["databaseVersion"])
		settings := postBody["settings"].(map[string]any)
		assert.Equal(t, "db-f1-micro", settings["tier"])
		assert.Equal(t, "10", settings["dataDiskSizeGb"])

		data := firstData(t, state)
		assert.Equal(t, "my-instance", data["name"])
		assert.Equal(t, "op-123", data["operation"])
		assert.Equal(t, "PENDING_CREATE", data["state"])
	})

	t.Run("missing name fails the execution", func(t *testing.T) {
		withFactory(&mockClient{projectID: "my-project"})
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"databaseVersion": "POSTGRES_16", "region": "us-central1", "tier": "db-f1-micro"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "name is required")
	})
}
