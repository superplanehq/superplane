package updatememory

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
	values        map[string]any
	updatedValues []any
	updateCalls   int
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

func (c *canvasMemoryContext) Update(namespace string, matches map[string]any, values map[string]any) ([]any, error) {
	c.updateCalls++
	c.namespace = namespace
	c.matches = matches
	c.values = values
	if c.err != nil {
		return nil, c.err
	}
	return c.updatedValues, nil
}

func TestUpdateMemoryExecute(t *testing.T) {
	t.Run("updates matches and emits found channel", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			updatedValues: []any{
				map[string]any{
					"creator":      "igor",
					"pull_request": 123,
					"sandbox_id":   "sbx-001",
					"status":       "running",
				},
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
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.updateCalls)
		assert.Equal(t, ChannelNameFound, execState.Channel)
		assert.Equal(
			t,
			map[string]any{
				"namespace":    "machines",
				"matchFields":  []string{"creator", "pull_request"},
				"valueFields":  []string{"status"},
				"matches":      map[string]any{"creator": "igor", "pull_request": 123},
				"updatedCount": 1,
			},
			nodeMetadata.Get(),
		)
	})

	t.Run("emits notFound channel when there are no matches", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{updatedValues: []any{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "nobody"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "ignored"},
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

	t.Run("returns error when update fails", func(t *testing.T) {
		component := &UpdateMemory{}
		memoryCtx := &canvasMemoryContext{err: errors.New("db failed")}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update canvas memory")
	})
}

func TestUpdateMemorySetup(t *testing.T) {
	component := &UpdateMemory{}

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
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
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
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
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one memory match is required")
	})

	t.Run("missing values fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "machines",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
				"valueList": []map[string]any{},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one memory value update is required")
	})
}
