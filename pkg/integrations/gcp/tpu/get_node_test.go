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

func TestGetNode_Metadata(t *testing.T) {
	c := &GetNode{}
	assert.Equal(t, "gcp.tpu.getNode", c.Name())
	assert.Equal(t, "Compute • Get TPU Node", c.Label())
	assert.NotEmpty(t, c.Documentation())

	output := c.ExampleOutput()
	assert.Equal(t, "gcp.tpu.node.read", output["type"])
}

func TestGetNode_Setup(t *testing.T) {
	c := &GetNode{}

	t.Run("valid config stores node name", func(t *testing.T) {
		meta := &testcontexts.MetadataContext{}
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{"node": "projects/my-project/locations/us-central1-b/nodes/my-tpu"},
			Metadata:      meta,
		})
		require.NoError(t, err)
		assert.Equal(t, "my-tpu", meta.Get().(TPUNodeMetadata).NodeName)
	})

	t.Run("missing node errors", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "node is required")
	})
}

func TestGetNode_Execute(t *testing.T) {
	t.Run("reads node and emits", func(t *testing.T) {
		var getURL string
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, url string) ([]byte, error) {
					getURL = url
					return nodeJSON("projects/my-project/locations/us-central1-b/nodes/my-tpu", "READY", nil), nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&GetNode{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"node": "projects/my-project/locations/us-central1-b/nodes/my-tpu"},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.tpu.node.read", state.Type)
		assert.True(t, strings.HasSuffix(getURL, "/locations/us-central1-b/nodes/my-tpu"))

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-tpu", data["name"])
		assert.Equal(t, "READY", data["state"])
	})

	t.Run("cross-project node fails before API call", func(t *testing.T) {
		var called bool
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					called = true
					return nil, nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&GetNode{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"node": "projects/other-project/locations/us-central1-b/nodes/my-tpu",
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
