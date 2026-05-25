package addmemory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type canvasMemoryContext struct {
	namespace string
	values    any
	addCalls  int
	addArgs   []any
	err       error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	c.addCalls++
	c.namespace = namespace
	c.values = values
	c.addArgs = append(c.addArgs, values)
	return c.err
}

func (c *canvasMemoryContext) Find(namespace string, matches map[string]any) ([]any, error) {
	return []any{}, c.err
}

func (c *canvasMemoryContext) FindFirst(namespace string, matches map[string]any) (any, error) {
	return nil, c.err
}

func TestAddMemoryExecute(t *testing.T) {
	t.Run("adds memory and emits payload", func(t *testing.T) {
		component := &AddMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{}
		execMetadata := &contexts.MetadataContext{}

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
			NodeMetadata:   &contexts.MetadataContext{},
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
			execMetadata.Get(),
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

	t.Run("list mode adds one row per element", func(t *testing.T) {
		component := &AddMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{}
		execMetadata := &contexts.MetadataContext{}

		items := []any{
			map[string]any{"name": "api", "id": "1"},
			map[string]any{"name": "worker", "id": "2"},
		}

		expressions := &contexts.ExpressionContext{
			Output: items,
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				item, ok := variables["item"].(map[string]any)
				require.True(t, ok)
				switch expression {
				case "item.name":
					return item["name"], nil
				case "item.id":
					return item["id"], nil
				}
				return nil, fmt.Errorf("unexpected expression %q", expression)
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   `$["Runner"].data.services`,
				"itemVariable": "item",
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
					{"name": "id", "value": "item.id"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions:    expressions,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.addCalls)
		assert.Equal(t, "services", memoryCtx.namespace)
		assert.Equal(t, []any{
			map[string]any{"name": "api", "id": "1"},
			map[string]any{"name": "worker", "id": "2"},
		}, memoryCtx.addArgs)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "services", metadata["namespace"])
		assert.Equal(t, []string{"name", "id"}, metadata["fields"])
		assert.Equal(t, true, metadata["iterateList"])
		assert.Equal(t, "item", metadata["itemVariable"])
		assert.Equal(t, 2, metadata["count"])

		require.Len(t, execState.Payloads, 1)
		payload, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		outerData, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "services", innerData["namespace"])
		assert.Equal(t, 2, innerData["count"])
		assert.Equal(t, []any{
			map[string]any{"name": "api", "id": "1"},
			map[string]any{"name": "worker", "id": "2"},
		}, innerData["values"])
	})

	t.Run("list mode rejects non-list source", func(t *testing.T) {
		component := &AddMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":   "services",
				"iterateList": true,
				"listSource":  "x",
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions:    &contexts.ExpressionContext{Output: "not a list"},
		})

		require.Error(t, err)
		assert.Equal(t, 0, memoryCtx.addCalls)
	})
}

func TestAddMemorySetup(t *testing.T) {
	t.Run("accepts valid list-mode config", func(t *testing.T) {
		err := (&AddMemory{}).Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   `$["Runner"].data.services`,
				"itemVariable": "service",
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects missing listSource", func(t *testing.T) {
		err := (&AddMemory{}).Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":   "services",
				"iterateList": true,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listSource")
	})

	t.Run("rejects reserved item variable", func(t *testing.T) {
		err := (&AddMemory{}).Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   "x",
				"itemVariable": "memory",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved")
	})
}
