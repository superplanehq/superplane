package readmemory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type canvasMemoryContext struct {
	namespace string
	matches   map[string]any
	values    []any
	findCalls int
	err       error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	return nil
}

func (c *canvasMemoryContext) Find(namespace string, matches map[string]any) ([]any, error) {
	c.findCalls++
	c.namespace = namespace
	c.matches = matches
	if c.err != nil {
		return nil, c.err
	}
	return c.values, nil
}

func (c *canvasMemoryContext) FindFirst(namespace string, matches map[string]any) (any, error) {
	return nil, c.err
}

func TestReadMemoryExecute(t *testing.T) {
	t.Run("reads memory and emits payload", func(t *testing.T) {
		component := &ReadMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			values: []any{
				map[string]any{"creator": "igor", "sandbox_id": "sbx-001"},
			},
		}
		execMetadata := &contexts.MetadataContext{}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
					{"name": "pull_request", "value": 123},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.findCalls)
		assert.Equal(t, "machines", memoryCtx.namespace)
		assert.Equal(t, map[string]any{"creator": "igor", "pull_request": 123}, memoryCtx.matches)

		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		assert.Equal(
			t,
			map[string]any{
				"namespace": "machines",
				"matches":   map[string]any{"creator": "igor", "pull_request": 123},
				"count":     1,
			},
			nodeMetadata.Get(),
		)
	})

	t.Run("returns error when namespace is empty", func(t *testing.T) {
		component := &ReadMemory{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": " ",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   &canvasMemoryContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("returns error when no matches are provided", func(t *testing.T) {
		component := &ReadMemory{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   &canvasMemoryContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one memory match is required")
	})

	t.Run("returns error when reading memory fails", func(t *testing.T) {
		component := &ReadMemory{}
		memoryCtx := &canvasMemoryContext{
			err: errors.New("db failed"),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read canvas memory")
	})
}

func TestReadMemorySetup(t *testing.T) {
	component := &ReadMemory{}

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("missing namespace fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("missing matches fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one memory match is required")
	})
}
