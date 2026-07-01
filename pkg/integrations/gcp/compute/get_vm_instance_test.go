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

type mockGetClient struct {
	projectID string
	getFunc   func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockGetClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockGetClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockGetClient) Delete(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockGetClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockGetClient) ProjectID() string {
	return m.projectID
}

// instanceResponse returns a minimal serialized instance GET response. Note
// that GCE's API returns the instance id as a string-encoded number, which the
// `instanceGetResp` struct deserializes via `json:"id,string"`.
func instanceResponse(name, zone, status string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id":          "12345",
		"name":        name,
		"status":      status,
		"zone":        fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/p/zones/%s", zone),
		"machineType": fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/p/zones/%s/machineTypes/f1-micro", zone),
		"selfLink":    fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/p/zones/%s/instances/%s", zone, name),
		"networkInterfaces": []map[string]any{
			{
				"networkIP": "10.0.0.5",
				"accessConfigs": []map[string]any{
					{"natIP": "35.1.2.3"},
				},
			},
		},
	})
	return b
}

func Test__GetVMInstance__Setup(t *testing.T) {
	component := &GetVMInstance{}

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

	t.Run("plain instance name is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.Error(t, err)
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

	t.Run("relative path stores parsed metadata", func(t *testing.T) {
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

	t.Run("selfLink URL stores parsed metadata", func(t *testing.T) {
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

func Test__GetVMInstance__Execute(t *testing.T) {
	component := &GetVMInstance{}

	t.Run("successful fetch -> emits instance payload", func(t *testing.T) {
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				assert.True(t, strings.HasSuffix(path, "/zones/us-central1-a/instances/my-vm"))
				return instanceResponse("my-vm", "us-central1-a", "RUNNING"), nil
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
		assert.Equal(t, "gcp.compute.vmInstance.fetched", state.Type)
		require.Len(t, state.Payloads, 1)
		wrapped := state.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "my-vm", data["name"])
		assert.Equal(t, "us-central1-a", data["zone"])
		assert.Equal(t, "RUNNING", data["status"])
		assert.Equal(t, "10.0.0.5", data["internalIP"])
		assert.Equal(t, "35.1.2.3", data["externalIP"])
		assert.Equal(t, "f1-micro", data["machineType"])
	})

	t.Run("selfLink form -> extracts zone and name", func(t *testing.T) {
		var capturedPath string
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				capturedPath = path
				return instanceResponse("my-vm", "us-central1-a", "RUNNING"), nil
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
		assert.True(t, state.Passed)
		assert.True(t, strings.Contains(capturedPath, "zones/us-central1-a/instances/my-vm"))
	})

	t.Run("instance not found (404) -> fails execution", func(t *testing.T) {
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
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

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.True(t, state.Finished)
		assert.Contains(t, state.FailureMessage, "failed to get VM instance")
	})

	t.Run("API error (not 404) -> fails execution", func(t *testing.T) {
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
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

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to get VM instance")
	})

	t.Run("unparseable response -> fails execution", func(t *testing.T) {
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
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

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "parse instance response")
	})

	t.Run("invalid instance value -> fails execution before any API call", func(t *testing.T) {
		var called bool
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return instanceResponse("my-vm", "us-central1-a", "RUNNING"), nil
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

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called, "Get API must not be called for an invalid instance value")
	})

	t.Run("cross-project selfLink -> fails execution before any API call", func(t *testing.T) {
		var called bool
		mc := &mockGetClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return instanceResponse("my-vm", "us-central1-a", "RUNNING"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) {
			return mc, nil
		})

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"instance": "https://www.googleapis.com/compute/v1/projects/other-project/zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called, "Get API must not be called when the URL project mismatches the integration")
		assert.Contains(t, state.FailureMessage, "other-project")
		assert.Contains(t, state.FailureMessage, "my-project")
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
