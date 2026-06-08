package compute

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// instanceNetworkJSON builds an instance GET response with a single nic0 holding
// the given access configs.
func instanceNetworkJSON(zone string, accessConfigs []map[string]any) []byte {
	nic := map[string]any{
		"name":      "nic0",
		"networkIP": "10.0.0.2",
	}
	if accessConfigs != nil {
		nic["accessConfigs"] = accessConfigs
	}
	b, _ := json.Marshal(map[string]any{
		"id":                "1234567890123456789",
		"name":              "my-vm",
		"selfLink":          "https://www.googleapis.com/compute/v1/projects/my-project/zones/" + zone + "/instances/my-vm",
		"status":            "RUNNING",
		"zone":              "https://www.googleapis.com/compute/v1/projects/my-project/zones/" + zone,
		"machineType":       "https://www.googleapis.com/compute/v1/projects/my-project/zones/" + zone + "/machineTypes/e2-medium",
		"networkInterfaces": []map[string]any{nic},
	})
	return b
}

func Test__ManageStaticIP__Setup(t *testing.T) {
	component := &ManageStaticIP{}

	t.Run("missing action returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "action is required")
	})

	t.Run("invalid action returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"action": "swap", "instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid action")
	})

	t.Run("attach without address returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"action": "attach", "instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "address is required")
	})

	t.Run("valid detach passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"action": "detach", "instance": "zones/us-central1-a/instances/my-vm"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__ManageStaticIP__Attach(t *testing.T) {
	component := &ManageStaticIP{}

	t.Run("attaches to interface with no existing external IP", func(t *testing.T) {
		var posts []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/addresses/"):
					return addressJSON("web-ip", "34.1.2.3", "us-central1", "RESERVED", "EXTERNAL", "PREMIUM"), nil
				default:
					return instanceNetworkJSON("us-central1-a", nil), nil
				}
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				posts = append(posts, path)
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "attach",
				"instance": "zones/us-central1-a/instances/my-vm",
				"address":  "regions/us-central1/addresses/web-ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.staticIP.attached", state.Type)
		require.Len(t, posts, 1)
		assert.Contains(t, posts[0], "addAccessConfig")
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "attach", data["action"])
	})

	t.Run("replaces an existing external IP (delete then add)", func(t *testing.T) {
		var posts []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/addresses/"):
					return addressJSON("web-ip", "34.1.2.3", "us-central1", "RESERVED", "EXTERNAL", "PREMIUM"), nil
				default:
					return instanceNetworkJSON("us-central1-a", []map[string]any{
						{"name": "External NAT", "type": "ONE_TO_ONE_NAT", "natIP": "34.0.0.9"},
					}), nil
				}
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				posts = append(posts, path)
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "attach",
				"instance": "zones/us-central1-a/instances/my-vm",
				"address":  "regions/us-central1/addresses/web-ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Len(t, posts, 2)
		assert.Contains(t, posts[0], "deleteAccessConfig")
		assert.Contains(t, posts[1], "addAccessConfig")
	})

	t.Run("idempotent when static IP already attached (no API writes)", func(t *testing.T) {
		var posts []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/addresses/"):
					return addressJSON("web-ip", "34.1.2.3", "us-central1", "RESERVED", "EXTERNAL", "PREMIUM"), nil
				default:
					return instanceNetworkJSON("us-central1-a", []map[string]any{
						{"name": "External NAT", "type": "ONE_TO_ONE_NAT", "natIP": "34.1.2.3"},
					}), nil
				}
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				posts = append(posts, path)
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "attach",
				"instance": "zones/us-central1-a/instances/my-vm",
				"address":  "regions/us-central1/addresses/web-ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Empty(t, posts, "no writes expected when already attached")
	})

	t.Run("region mismatch -> fails", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return instanceNetworkJSON("us-central1-a", nil), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "attach",
				"instance": "zones/us-central1-a/instances/my-vm",
				"address":  "regions/europe-west1/addresses/web-ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "same region")
	})
}

func Test__ManageStaticIP__Detach(t *testing.T) {
	component := &ManageStaticIP{}

	t.Run("detaches an existing external IP", func(t *testing.T) {
		var posts []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op"), nil
				}
				return instanceNetworkJSON("us-central1-a", []map[string]any{
					{"name": "External NAT", "type": "ONE_TO_ONE_NAT", "natIP": "34.0.0.9"},
				}), nil
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				posts = append(posts, path)
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "detach",
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.staticIP.detached", state.Type)
		require.Len(t, posts, 1)
		assert.Contains(t, posts[0], "deleteAccessConfig")
	})

	t.Run("idempotent when no external IP present (no API writes)", func(t *testing.T) {
		var posts []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op"), nil
				}
				return instanceNetworkJSON("us-central1-a", nil), nil
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				posts = append(posts, path)
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"action":   "detach",
				"instance": "zones/us-central1-a/instances/my-vm",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Empty(t, posts)
	})
}
