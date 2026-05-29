package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ManageVMInstancePower__Setup(t *testing.T) {
	component := &ManageVMInstancePower{}

	t.Run("missing instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"operation": "power_on"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("missing operation returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "operation is required")
	})

	t.Run("invalid operation returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance":  "zones/us-central1-a/instances/my-vm",
				"operation": "explode",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("valid config stores parsed metadata", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance":  "zones/us-central1-a/instances/my-vm",
				"operation": "power_off",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
	})
}

func Test__ManageVMInstancePower__Execute(t *testing.T) {
	component := &ManageVMInstancePower{}

	t.Run("power_off -> stops instance and emits power event", func(t *testing.T) {
		var postedPath string
		mc := &mockInstanceClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postedPath = path
				return opDone("op-1"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-1"), nil
				}
				return instanceGetJSON("123", "my-vm", "us-central1-a", "TERMINATED", "e2-medium"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":  "zones/us-central1-a/instances/my-vm",
				"operation": "power_off",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.True(t, strings.HasSuffix(postedPath, "/zones/us-central1-a/instances/my-vm/stop"))
		assert.Equal(t, "gcp.compute.vmInstance.power.power_off", state.Type)
		require.Len(t, state.Payloads, 1)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-vm", data["name"])
		assert.Equal(t, "power_off", data["operation"])
		assert.Equal(t, "TERMINATED", data["status"])
	})

	t.Run("power_on -> uses start endpoint", func(t *testing.T) {
		var postedPath string
		mc := &mockInstanceClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postedPath = path
				return opDone("op-2"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-2"), nil
				}
				return instanceGetJSON("123", "my-vm", "us-central1-a", "RUNNING", "e2-medium"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":  "zones/us-central1-a/instances/my-vm",
				"operation": "power_on",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.True(t, strings.HasSuffix(postedPath, "/start"))
		assert.Equal(t, "gcp.compute.vmInstance.power.power_on", state.Type)
	})

	t.Run("cross-project selfLink -> fails", func(t *testing.T) {
		mc := &mockInstanceClient{projectID: "my-project"}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":  "https://www.googleapis.com/compute/v1/projects/other/zones/us-central1-a/instances/my-vm",
				"operation": "power_off",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
