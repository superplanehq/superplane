package cloudsql

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteInstance__Setup(t *testing.T) {
	d := &DeleteInstance{}
	setup := func(cfg map[string]any) error {
		return d.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}

	t.Run("missing instance -> error", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{}), "instance is required")
	})

	t.Run("valid -> ok", func(t *testing.T) {
		require.NoError(t, setup(map[string]any{"instance": "my-instance"}))
	})
}

func Test__DeleteInstance__Execute(t *testing.T) {
	d := &DeleteInstance{}

	t.Run("deletes the instance and emits the operation", func(t *testing.T) {
		var deleteURL string
		mc := &mockClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, url string) ([]byte, error) {
				deleteURL = url
				return []byte(`{"name":"op-456","status":"PENDING","targetId":"my-instance"}`), nil
			},
		}
		withFactory(mc)

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"instance": "my-instance"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.cloudsql.instance", state.Type)
		assert.True(t, strings.HasSuffix(deleteURL, "/projects/my-project/instances/my-instance"))

		data := firstData(t, state)
		assert.Equal(t, "my-instance", data["name"])
		assert.Equal(t, "op-456", data["operation"])
		assert.Equal(t, true, data["deleting"])
	})
}
