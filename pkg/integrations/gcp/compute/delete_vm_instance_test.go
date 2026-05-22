package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type mockDeleteClient struct {
	projectID  string
	getFunc    func(ctx context.Context, path string) ([]byte, error)
	deleteFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockDeleteClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDeleteClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDeleteClient) Delete(ctx context.Context, path string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDeleteClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDeleteClient) ProjectID() string {
	return m.projectID
}

// opDone returns a serialized DONE zone operation response
func opDone(name string) []byte {
	b, _ := json.Marshal(map[string]any{"name": name, "status": "DONE"})
	return b
}

func Test__DeleteVMInstance__Setup(t *testing.T) {
	component := &DeleteVMInstance{}

	t.Run("missing zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance": "my-vm",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone": "us-central1-a",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("empty zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "",
				"instance": "my-vm",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("empty instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("expression instance is accepted without API call", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "{{ $.trigger.data.instanceName }}",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored VMInstanceNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "{{ $.trigger.data.instanceName }}", stored.InstanceName)
		assert.Equal(t, "us-central1-a", stored.Zone)
	})

	t.Run("valid config without integration stores metadata", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "my-vm",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored VMInstanceNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "my-vm", stored.InstanceName)
		assert.Equal(t, "us-central1-a", stored.Zone)
	})
}

func Test__DeleteVMInstance__Execute(t *testing.T) {
	component := &DeleteVMInstance{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				assert.True(t, strings.HasSuffix(path, "/instances/my-vm"))
				return opDone("operation-123"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				// Poll for operation status
				return opDone("operation-123"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "default", state.Channel)
		assert.Equal(t, "gcp.compute.vmInstance.deleted", state.Type)
		require.Len(t, state.Payloads, 1)
		wrapped := state.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "my-vm", data["instanceName"])
		assert.Equal(t, "us-central1-a", data["zone"])
	})

	t.Run("instance not found (404) -> emits success (idempotent)", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "Instance not found"}
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.vmInstance.deleted", state.Type)
	})

	t.Run("API error (not 404) -> returns error", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusInternalServerError, Message: "internal error"}
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":     "us-central1-a",
				"instance": "my-vm",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete VM instance")
		assert.False(t, state.Passed)
	})

	t.Run("zone last-segment is extracted correctly", func(t *testing.T) {
		var capturedPath string
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				capturedPath = path
				return opDone("operation-abc"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return opDone("operation-abc"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				// zone as full resource URL
				"zone":     "projects/my-project/zones/us-central1-a",
				"instance": "my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, strings.Contains(capturedPath, "zones/us-central1-a/instances/my-vm"))
	})
}
