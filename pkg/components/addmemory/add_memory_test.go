package addmemory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type canvasMemoryContext struct {
	namespace string
	values    any
	addCalls  int
	err       error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	c.addCalls++
	c.namespace = namespace
	c.values = values
	return c.err
}

func TestAddMemoryExecute(t *testing.T) {
	t.Run("adds memory and emits payload", func(t *testing.T) {
		component := &AddMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{}
		execMetadata := &contexts.MetadataContext{}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"valueList": []map[string]any{
					{"name": "id", "value": "1"},
					{"name": "pull_request", "value": "123"},
					{"name": "creator", "value": "alex"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.addCalls)
		assert.Equal(t, "machines", memoryCtx.namespace)
		assert.Equal(
			t,
			map[string]any{"id": "1", "pull_request": "123", "creator": "alex"},
			memoryCtx.values,
		)
		assert.Equal(
			t,
			map[string]any{
				"namespace": "machines",
				"fields":    []string{"id", "pull_request", "creator"},
			},
			nodeMetadata.Get(),
		)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)
		emittedPayload, ok := execState.Payloads[0].(map[string]any)
		assert.True(t, ok)
		outerData, ok := emittedPayload["data"].(map[string]any)
		assert.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "machines", innerData["namespace"])
		assert.Equal(
			t,
			map[string]any{"id": "1", "pull_request": "123", "creator": "alex"},
			innerData["values"],
		)
	})

}
