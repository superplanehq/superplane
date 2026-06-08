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

func Test__UpdateVMInstanceType__Setup(t *testing.T) {
	component := &UpdateVMInstanceType{}

	t.Run("missing instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"machineType": "e2-medium"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("missing machineType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "machineType is required")
	})
}

func Test__UpdateVMInstanceType__Execute(t *testing.T) {
	component := &UpdateVMInstanceType{}

	t.Run("running instance -> stops, sets type, restarts, emits", func(t *testing.T) {
		var postedPaths []string
		var setMachineTypeBody map[string]any
		getCalls := 0
		mc := &mockInstanceClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postedPaths = append(postedPaths, path)
				if strings.HasSuffix(path, "/setMachineType") {
					setMachineTypeBody, _ = body.(map[string]any)
				}
				return opDone("op-1"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-1"), nil
				}
				getCalls++
				// First read: RUNNING (forces a stop). Final read: new type.
				if getCalls == 1 {
					return instanceGetJSON("123", "my-vm", "us-central1-a", "RUNNING", "e2-medium"), nil
				}
				return instanceGetJSON("123", "my-vm", "us-central1-a", "RUNNING", "n2-standard-4"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":    "zones/us-central1-a/instances/my-vm",
				"machineType": "n2-standard-4",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.vmInstance.machineTypeUpdated", state.Type)

		// stop, setMachineType, start (in order)
		require.Len(t, postedPaths, 3)
		assert.True(t, strings.HasSuffix(postedPaths[0], "/stop"))
		assert.True(t, strings.HasSuffix(postedPaths[1], "/setMachineType"))
		assert.True(t, strings.HasSuffix(postedPaths[2], "/start"))
		assert.Equal(t, "zones/us-central1-a/machineTypes/n2-standard-4", setMachineTypeBody["machineType"])

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "n2-standard-4", data["machineType"])
	})

	t.Run("already stopped + no restart -> skips stop and start", func(t *testing.T) {
		var postedPaths []string
		restart := false
		mc := &mockInstanceClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postedPaths = append(postedPaths, path)
				return opDone("op-1"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-1"), nil
				}
				return instanceGetJSON("123", "my-vm", "us-central1-a", "TERMINATED", "n2-standard-4"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance":           "zones/us-central1-a/instances/my-vm",
				"machineType":        "n2-standard-4",
				"restartAfterUpdate": restart,
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, postedPaths, 1)
		assert.True(t, strings.HasSuffix(postedPaths[0], "/setMachineType"))
	})
}
