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

func Test__ParseInstancePath(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		zone, name, err := parseInstancePath("zones/us-central1-a/instances/my-vm")
		require.NoError(t, err)
		assert.Equal(t, "us-central1-a", zone)
		assert.Equal(t, "my-vm", name)
	})

	t.Run("full selfLink URL", func(t *testing.T) {
		selfLink := "https://www.googleapis.com/compute/v1/projects/elffie/zones/europe-west1-b/instances/web-server-01"
		zone, name, err := parseInstancePath(selfLink)
		require.NoError(t, err)
		assert.Equal(t, "europe-west1-b", zone)
		assert.Equal(t, "web-server-01", name)
	})

	t.Run("project-qualified relative path", func(t *testing.T) {
		zone, name, err := parseInstancePath("projects/elffie/zones/us-east1-c/instances/db-1")
		require.NoError(t, err)
		assert.Equal(t, "us-east1-c", zone)
		assert.Equal(t, "db-1", name)
	})

	t.Run("trims surrounding whitespace", func(t *testing.T) {
		zone, name, err := parseInstancePath("  zones/us-central1-a/instances/my-vm  ")
		require.NoError(t, err)
		assert.Equal(t, "us-central1-a", zone)
		assert.Equal(t, "my-vm", name)
	})

	t.Run("plain name is rejected", func(t *testing.T) {
		_, _, err := parseInstancePath("just-a-name")
		require.Error(t, err)
	})

	t.Run("empty value is rejected", func(t *testing.T) {
		_, _, err := parseInstancePath("")
		require.Error(t, err)
	})

	t.Run("missing instances segment is rejected", func(t *testing.T) {
		_, _, err := parseInstancePath("zones/us-central1-a/foo/my-vm")
		require.Error(t, err)
	})
}

func Test__DeleteVMInstance__Setup(t *testing.T) {
	component := &DeleteVMInstance{}

	t.Run("missing instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("empty instance returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": ""},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "instance is required")
	})

	t.Run("plain instance name is rejected (missing zone segment)", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "zones/")
	})

	t.Run("expression instance is accepted without API call", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance": "{{ $.nodes.create.outputs.default[0].data.selfLink }}",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored VMInstanceNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "{{ $.nodes.create.outputs.default[0].data.selfLink }}", stored.InstanceName)
	})

	t.Run("relative path without integration stores parsed metadata", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored VMInstanceNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "my-vm", stored.InstanceName)
		assert.Equal(t, "us-central1-a", stored.Zone)
	})

	t.Run("selfLink URL without integration stores parsed metadata", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"instance": "https://www.googleapis.com/compute/v1/projects/elffie/zones/europe-west1-b/instances/db-1",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored VMInstanceNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "db-1", stored.InstanceName)
		assert.Equal(t, "europe-west1-b", stored.Zone)
	})
}

func Test__DeleteVMInstance__Execute(t *testing.T) {
	component := &DeleteVMInstance{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				assert.True(t, strings.HasSuffix(path, "/zones/us-central1-a/instances/my-vm"))
				return opDone("operation-123"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return opDone("operation-123"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance": "zones/us-central1-a/instances/my-vm",
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

	t.Run("selfLink form -> extracts zone and name", func(t *testing.T) {
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
				"instance": "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, strings.Contains(capturedPath, "zones/us-central1-a/instances/my-vm"))
	})

	t.Run("instance not found (404) -> returns error (no silent success)", func(t *testing.T) {
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
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete VM instance")
		assert.False(t, state.Passed)
	})

	t.Run("unparseable delete response -> returns error", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return []byte("not-json"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse delete operation response")
		assert.False(t, state.Passed)
	})

	t.Run("delete response missing operation name -> returns error", func(t *testing.T) {
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				body, _ := json.Marshal(map[string]any{"status": "PENDING"})
				return body, nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing operation name")
		assert.False(t, state.Passed)
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
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete VM instance")
		assert.False(t, state.Passed)
	})

	t.Run("invalid instance value -> returns error before any API call", func(t *testing.T) {
		var called bool
		mc := &mockDeleteClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return opDone("op-x"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance": "just-a-name",
			},
			ExecutionState: state,
		})

		require.Error(t, err)
		assert.False(t, called, "Delete API must not be called for an invalid instance value")
	})
}
