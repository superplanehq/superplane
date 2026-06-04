package tpu

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateNode_Metadata(t *testing.T) {
	c := &CreateNode{}
	assert.Equal(t, "gcp.tpu.createNode", c.Name())
	assert.Equal(t, "Compute • Create TPU Node", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.NotEmpty(t, c.Documentation())

	output := c.ExampleOutput()
	assert.Equal(t, "gcp.tpu.node.created", output["type"])
	assert.NotEmpty(t, output["timestamp"])
	assert.NotNil(t, output["data"])
}

func TestCreateNode_Setup(t *testing.T) {
	c := &CreateNode{}

	t.Run("valid config stores node name", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":            "my-tpu",
				"location":        "us-central1-b",
				"acceleratorType": "v2-8",
				"runtimeVersion":  "tpu-vm-tf-2.16.1",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		assert.Equal(t, "my-tpu", meta.Get().(TPUNodeMetadata).NodeName)
	})

	t.Run("missing fields error", func(t *testing.T) {
		cases := []struct {
			name string
			cfg  map[string]any
			want string
		}{
			{"no name", map[string]any{"location": "us-central1-b", "acceleratorType": "v2-8", "runtimeVersion": "rv"}, "node name is required"},
			{"no location", map[string]any{"name": "my-tpu", "acceleratorType": "v2-8", "runtimeVersion": "rv"}, "location is required"},
			{"no accelerator", map[string]any{"name": "my-tpu", "location": "us-central1-b", "runtimeVersion": "rv"}, "accelerator type is required"},
			{"no runtime", map[string]any{"name": "my-tpu", "location": "us-central1-b", "acceleratorType": "v2-8"}, "runtime version is required"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := c.Setup(core.SetupContext{Configuration: tc.cfg, Metadata: &testcontexts.MetadataContext{}})
				require.ErrorContains(t, err, tc.want)
			})
		}
	})
}

func TestCreateNode_Execute(t *testing.T) {
	t.Run("creates node, waits, and emits", func(t *testing.T) {
		var postURL string
		var postBody any
		opName := "projects/my-project/locations/us-central1-b/operations/op1"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, url string, body any) ([]byte, error) {
					postURL = url
					postBody = body
					return opStartedJSON(opName), nil
				},
				getURL: func(_ context.Context, url string) ([]byte, error) {
					if isOperationURL(url) {
						return opDoneJSON(opName), nil
					}
					return nodeJSON("projects/my-project/locations/us-central1-b/nodes/my-tpu", "READY", map[string]string{"env": "prod"}), nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&CreateNode{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":            "my-tpu",
				"location":        "us-central1-b",
				"acceleratorType": "v2-8",
				"runtimeVersion":  "tpu-vm-tf-2.16.1",
				"preemptible":     true,
				"labels":          []any{map[string]any{"key": "env", "value": "prod"}},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.tpu.node.created", state.Type)
		assert.True(t, strings.Contains(postURL, "/locations/us-central1-b/nodes?nodeId=my-tpu"))

		node := postBody.(*Node)
		assert.Equal(t, "v2-8", node.AcceleratorType)
		assert.Equal(t, "tpu-vm-tf-2.16.1", node.RuntimeVersion)
		require.NotNil(t, node.SchedulingConfig)
		assert.True(t, node.SchedulingConfig.Preemptible)
		assert.Equal(t, map[string]string{"env": "prod"}, node.Labels)

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-tpu", data["name"])
		assert.Equal(t, "READY", data["state"])
	})

	t.Run("create API error fails execution", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return []byte("not-json"), nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&CreateNode{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":            "my-tpu",
				"location":        "us-central1-b",
				"acceleratorType": "v2-8",
				"runtimeVersion":  "tpu-vm-tf-2.16.1",
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "create node operation response")
	})
}

func TestCreateNode_BuildNodeFromSpec(t *testing.T) {
	t.Run("omits network config when unset", func(t *testing.T) {
		node := buildNodeFromSpec(CreateNodeSpec{AcceleratorType: "v2-8", RuntimeVersion: "rv"})
		assert.Nil(t, node.NetworkConfig)
		assert.Nil(t, node.SchedulingConfig)
	})

	t.Run("sets network config when network provided", func(t *testing.T) {
		node := buildNodeFromSpec(CreateNodeSpec{AcceleratorType: "v2-8", RuntimeVersion: "rv", Network: "default", EnableExternalIps: true})
		require.NotNil(t, node.NetworkConfig)
		assert.Equal(t, "default", node.NetworkConfig.Network)
		assert.True(t, node.NetworkConfig.EnableExternalIps)
	})
}
