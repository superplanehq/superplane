package upsertmemory

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
	addCalls      int
	err           error
	addErr        error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	c.addCalls++
	c.namespace = namespace
	valueMap, ok := values.(map[string]any)
	if ok {
		c.values = valueMap
	}
	return c.addErr
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

func TestUpsertMemoryExecute(t *testing.T) {
	t.Run("updates matches and emits updated channel", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			updatedValues: []any{
				map[string]any{
					"environment":              "production",
					"latest_deployment":        "v1.0.1",
					"latest_deployment_source": "manual_run",
				},
			},
		}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.1"},
					{"name": "latest_deployment_source", "value": "manual_run"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.updateCalls)
		assert.Equal(t, 0, memoryCtx.addCalls)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(
			t,
			map[string]any{
				"namespace":    "deployments",
				"matchFields":  []string{"environment"},
				"valueFields":  []string{"latest_deployment", "latest_deployment_source"},
				"matches":      map[string]any{"environment": "production"},
				"operation":    OperationUpdated,
				"updatedCount": 1,
			},
			nodeMetadata.Get(),
		)
	})

	t.Run("creates row and emits created channel when no matches", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{updatedValues: []any{}}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "staging"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.2"},
					{"name": "latest_deployment_source", "value": "manual_run"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.updateCalls)
		assert.Equal(t, 1, memoryCtx.addCalls)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(
			t,
			map[string]any{
				"namespace":    "deployments",
				"matchFields":  []string{"environment"},
				"valueFields":  []string{"latest_deployment", "latest_deployment_source"},
				"matches":      map[string]any{"environment": "staging"},
				"operation":    OperationCreated,
				"updatedCount": 0,
			},
			nodeMetadata.Get(),
		)
	})

	t.Run("supports empty matches for namespace-level upsert", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			updatedValues: []any{
				map[string]any{
					"value": "new-sha",
				},
			},
		}
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"valueList": []map[string]any{
					{"name": "value", "value": "new-sha"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   nodeMetadata,
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, memoryCtx.updateCalls)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, map[string]any{}, memoryCtx.matches)
		assert.Equal(t, map[string]any{}, nodeMetadata.Get().(map[string]any)["matches"])
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		component := &UpsertMemory{}
		memoryCtx := &canvasMemoryContext{err: errors.New("db failed")}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.1"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert canvas memory")
	})

	t.Run("returns error when add fails after not found", func(t *testing.T) {
		component := &UpsertMemory{}
		memoryCtx := &canvasMemoryContext{
			updatedValues: []any{},
			addErr:        errors.New("insert failed"),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.1"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert canvas memory")
	})
}

func TestUpsertMemorySetup(t *testing.T) {
	component := &UpsertMemory{}

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.1"},
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
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "latest_deployment", "value": "v1.0.1"},
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "namespace is required")
	})

	t.Run("missing values fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one memory value update is required")
	})

	t.Run("empty matches are allowed", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"matchList": []map[string]any{},
				"valueList": []map[string]any{
					{"name": "value", "value": "abc123"},
				},
			},
		})
		assert.NoError(t, err)
	})
}
