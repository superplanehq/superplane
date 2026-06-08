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

func TestDeleteNode_Metadata(t *testing.T) {
	c := &DeleteNode{}
	assert.Equal(t, "gcp.tpu.deleteNode", c.Name())
	assert.Equal(t, "Compute • Delete TPU Node", c.Label())
	assert.NotEmpty(t, c.Documentation())

	output := c.ExampleOutput()
	assert.Equal(t, "gcp.tpu.node.deleted", output["type"])
}

func TestDeleteNode_Setup(t *testing.T) {
	c := &DeleteNode{}
	meta := &testcontexts.MetadataContext{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{"node": "projects/my-project/locations/us-central1-b/nodes/my-tpu"},
		Metadata:      meta,
	})
	require.NoError(t, err)
	assert.Equal(t, "my-tpu", meta.Get().(TPUNodeMetadata).NodeName)
}

func TestDeleteNode_Execute(t *testing.T) {
	t.Run("deletes node, waits, and emits", func(t *testing.T) {
		var deleteURL string
		opName := "projects/my-project/locations/us-central1-b/operations/op1"
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				deleteURL: func(_ context.Context, url string) ([]byte, error) {
					deleteURL = url
					return opStartedJSON(opName), nil
				},
				getURL: func(_ context.Context, _ string) ([]byte, error) {
					return opDoneJSON(opName), nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&DeleteNode{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"node": "projects/my-project/locations/us-central1-b/nodes/my-tpu"},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.tpu.node.deleted", state.Type)
		assert.True(t, strings.HasSuffix(deleteURL, "/locations/us-central1-b/nodes/my-tpu"))

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-tpu", data["name"])
		assert.Equal(t, "us-central1-b", data["location"])
	})

	t.Run("delete API error fails execution", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				deleteURL: func(_ context.Context, _ string) ([]byte, error) {
					return []byte("not-json"), nil
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&DeleteNode{}).Execute(core.ExecutionContext{
			Configuration:  map[string]any{"node": "projects/my-project/locations/us-central1-b/nodes/my-tpu"},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "delete node operation response")
	})
}
