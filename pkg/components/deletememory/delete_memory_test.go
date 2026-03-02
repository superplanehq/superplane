package deletememory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type canvasMemoryContext struct {
	namespace     string
	matches       map[string]any
	deletedValues []any
	deleteCalls   int
	err           error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	return nil
}

func (c *canvasMemoryContext) Find(namespace string, matches map[string]any) ([]any, error) {
	return []any{}, nil
}

func (c *canvasMemoryContext) FindFirst(namespace string, matches map[string]any) (any, error) {
	return nil, nil
}

func (c *canvasMemoryContext) Delete(namespace string, matches map[string]any) ([]any, error) {
	c.deleteCalls++
	c.namespace = namespace
	c.matches = matches
	if c.err != nil {
		return nil, c.err
	}
	return c.deletedValues, nil
}

func TestDeleteMemoryExecute(t *testing.T) {
	t.Run("deletes all matches and emits payload", func(t *testing.T) {
		component := &DeleteMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			deletedValues: []any{
				map[string]any{"creator": "igor", "sandbox_id": "sbx-001"},
			},
		}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
					{"name": "pull_request", "value": 123},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.deleteCalls)
		assert.Equal(t, ChannelNameDeleted, execState.Channel)
		assert.Equal(
			t,
			map[string]any{
				"namespace": "machines",
				"fields":    []string{"creator", "pull_request"},
				"matches":   map[string]any{"creator": "igor", "pull_request": 123},
				"count":     1,
			},
			nodeMetadata.Get(),
		)
	})

	t.Run("emits notFound channel when no rows are removed", func(t *testing.T) {
		component := &DeleteMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{deletedValues: []any{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "nobody"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, ChannelNameNotFound, execState.Channel)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		component := &DeleteMemory{}
		memoryCtx := &canvasMemoryContext{err: errors.New("db failed")}

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
		assert.Contains(t, err.Error(), "failed to delete canvas memory")
	})
}

func TestDeleteMemorySetup(t *testing.T) {
	component := &DeleteMemory{}

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
