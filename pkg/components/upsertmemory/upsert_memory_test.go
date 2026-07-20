package upsertmemory

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type canvasMemoryContext struct {
	namespace              string
	matches                map[string]any
	values                 map[string]any
	addedValues            []any
	updatedValues          []any
	namespaceUpdatedValues []any
	updatedPerCall         [][]any
	updatedRecordsPerCall  [][]core.CanvasMemoryRecord
	updateCalls            int
	namespaceUpdateCalls   int
	addCalls               int
	err                    error
	addErr                 error
}

func (c *canvasMemoryContext) Add(namespace string, values any) error {
	c.addCalls++
	c.namespace = namespace
	valueMap, ok := values.(map[string]any)
	if ok {
		c.values = valueMap
	}
	c.addedValues = append(c.addedValues, values)
	return c.addErr
}

func (c *canvasMemoryContext) AddRecord(namespace string, values any) (core.CanvasMemoryRecord, error) {
	err := c.Add(namespace, values)
	if err != nil {
		return core.CanvasMemoryRecord{}, err
	}

	return core.CanvasMemoryRecord{
		ID:     uuid.New(),
		Values: values,
	}, nil
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
	if len(matches) == 0 {
		return nil, errors.New("at least one match expression is required")
	}
	if len(c.updatedPerCall) > 0 {
		idx := c.updateCalls - 1
		if idx >= len(c.updatedPerCall) {
			idx = len(c.updatedPerCall) - 1
		}
		return c.updatedPerCall[idx], nil
	}
	return c.updatedValues, nil
}

func (c *canvasMemoryContext) UpdateRecords(namespace string, matches map[string]any, values map[string]any) ([]core.CanvasMemoryRecord, error) {
	c.updateCalls++
	c.namespace = namespace
	c.matches = matches
	c.values = values
	if c.err != nil {
		return nil, c.err
	}
	if len(matches) == 0 {
		return nil, errors.New("at least one match expression is required")
	}
	if len(c.updatedRecordsPerCall) == 0 {
		return nil, nil
	}

	idx := c.updateCalls - 1
	if idx >= len(c.updatedRecordsPerCall) {
		idx = len(c.updatedRecordsPerCall) - 1
	}
	return c.updatedRecordsPerCall[idx], nil
}

func (c *canvasMemoryContext) UpdateNamespace(namespace string, values map[string]any) ([]any, error) {
	c.namespaceUpdateCalls++
	c.namespace = namespace
	c.values = values
	if c.err != nil {
		return nil, c.err
	}
	if c.namespaceUpdatedValues != nil {
		return c.namespaceUpdatedValues, nil
	}
	return c.updatedValues, nil
}

func (c *canvasMemoryContext) UpdateNamespaceRecords(namespace string, values map[string]any) ([]core.CanvasMemoryRecord, error) {
	c.namespaceUpdateCalls++
	c.namespace = namespace
	c.values = values
	if c.err != nil {
		return nil, c.err
	}
	return nil, nil
}

func memoryRecord(id string, values any) core.CanvasMemoryRecord {
	return core.CanvasMemoryRecord{
		ID:     uuid.MustParse(id),
		Values: values,
	}
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
		execMetadata := &contexts.MetadataContext{}

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
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
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
			execMetadata.Get(),
		)
	})

	t.Run("creates row and emits created channel when no matches", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{updatedValues: []any{}}
		execMetadata := &contexts.MetadataContext{}

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
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
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
			execMetadata.Get(),
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
		execMetadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace": "deployments",
				"valueList": []map[string]any{
					{"name": "value", "value": "new-sha"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.Equal(t, 0, memoryCtx.updateCalls)
		assert.Equal(t, 1, memoryCtx.namespaceUpdateCalls)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, map[string]any{"value": "new-sha"}, memoryCtx.values)
		assert.Equal(t, map[string]any{}, execMetadata.Get().(map[string]any)["matches"])
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

	t.Run("list mode upserts one row per element", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			updatedRecordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"environment": "production", "name": "api"})},
			},
		}
		execMetadata := &contexts.MetadataContext{}

		items := []any{
			map[string]any{"name": "api"},
			map[string]any{"name": "worker"},
		}

		expressions := &contexts.ExpressionContext{
			Output: items,
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				return variables["item"].(map[string]any)["name"], nil
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "deployments",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList":    []map[string]any{},
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions:    expressions,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, memoryCtx.updateCalls)
		assert.Equal(t, 0, memoryCtx.namespaceUpdateCalls)
		assert.Equal(t, 2, memoryCtx.addCalls)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, metadata["iterateList"])
		assert.Equal(t, "item", metadata["itemVariable"])
		assert.Equal(t, 2, metadata["count"])
		assert.Equal(t, 0, metadata["updatedCount"])
		assert.Equal(t, 2, metadata["createdCount"])

		require.Len(t, execState.Payloads, 1)
		payload, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		outerData, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		require.True(t, ok)

		itemsResults, ok := innerData["items"].([]any)
		require.True(t, ok)
		require.Len(t, itemsResults, 2)
		first, ok := itemsResults[0].(map[string]any)
		require.True(t, ok)
		second, ok := itemsResults[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, OperationCreated, first["operation"])
		assert.Equal(t, OperationCreated, second["operation"])
	})

	t.Run("list mode with matches updates then creates per item", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			updatedRecordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"environment": "production", "name": "api"})},
				nil,
			},
		}
		execMetadata := &contexts.MetadataContext{}

		items := []any{
			map[string]any{"name": "api"},
			map[string]any{"name": "worker"},
		}

		expressions := &contexts.ExpressionContext{
			Output: items,
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				return variables["item"].(map[string]any)["name"], nil
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "deployments",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions:    expressions,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.updateCalls)
		assert.Equal(t, 1, memoryCtx.addCalls)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, metadata["updatedCount"])
		assert.Equal(t, 1, metadata["createdCount"])
	})

	t.Run("list mode deduplicates repeated updated record", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{
			updatedRecordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"environment": "production", "name": "api"})},
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"environment": "production", "name": "worker"})},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "deployments",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "environment", "value": "production"},
				},
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: []any{
					map[string]any{"name": "api"},
					map[string]any{"name": "worker"},
				},
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					return variables["item"].(map[string]any)["name"], nil
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.updateCalls)
		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, metadata["updatedCount"])
		assert.Equal(t, 0, metadata["createdCount"])

		require.Len(t, execState.Payloads, 1)
		payload, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		outerData, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, innerData["count"])
		assert.Equal(t, []any{
			map[string]any{"environment": "production", "name": "worker"},
		}, innerData["records"])
	})

	t.Run("list mode resolves matchList per item with iteration variable", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{
			updatedRecordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"uuid": "A", "name": "api"})},
				nil,
			},
		}

		items := []any{
			map[string]any{"uuid": "A", "name": "api"},
			map[string]any{"uuid": "B", "name": "worker"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "apps",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "uuid", "value": "item.uuid"},
				},
				"valueList": []map[string]any{
					{"name": "uuid", "value": "item.uuid"},
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: items,
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					item := variables["item"].(map[string]any)
					switch expression {
					case "item.uuid":
						return item["uuid"], nil
					case "item.name":
						return item["name"], nil
					}
					return expression, nil
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, 2, memoryCtx.updateCalls)
		assert.Equal(t, 1, memoryCtx.addCalls)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, metadata["updatedCount"])
		assert.Equal(t, 1, metadata["createdCount"])

		require.Len(t, execState.Payloads, 1)
		payload, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		outerData, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		require.True(t, ok)
		itemsResults, ok := innerData["items"].([]any)
		require.True(t, ok)
		require.Len(t, itemsResults, 2)
		first, ok := itemsResults[0].(map[string]any)
		require.True(t, ok)
		second, ok := itemsResults[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, map[string]any{"uuid": "A"}, first["matches"])
		assert.Equal(t, map[string]any{"uuid": "B"}, second["matches"])
		assert.Equal(t, OperationUpdated, first["operation"])
		assert.Equal(t, OperationCreated, second["operation"])
	})

	t.Run("list mode upserts existing rows on second run via per-item match", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{
			updatedRecordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"uuid": "A", "name": "api"})},
				{memoryRecord("22222222-2222-2222-2222-222222222222", map[string]any{"uuid": "B", "name": "worker"})},
			},
		}

		items := []any{
			map[string]any{"uuid": "A", "name": "api"},
			map[string]any{"uuid": "B", "name": "worker"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "apps",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "uuid", "value": "item.uuid"},
				},
				"valueList": []map[string]any{
					{"name": "uuid", "value": "item.uuid"},
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: items,
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					item := variables["item"].(map[string]any)
					switch expression {
					case "item.uuid":
						return item["uuid"], nil
					case "item.name":
						return item["name"], nil
					}
					return expression, nil
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.updateCalls)
		assert.Equal(t, 0, memoryCtx.addCalls)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, metadata["updatedCount"])
		assert.Equal(t, 0, metadata["createdCount"])
	})

	t.Run("list mode treats blank-name matchList as insert-only", func(t *testing.T) {
		component := &UpsertMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{}

		items := []any{
			map[string]any{"uuid": "A", "name": "api"},
			map[string]any{"uuid": "B", "name": "worker"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "apps",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "", "value": "item.uuid"},
				},
				"valueList": []map[string]any{
					{"name": "uuid", "value": "item.uuid"},
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: items,
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					item := variables["item"].(map[string]any)
					switch expression {
					case "item.uuid":
						return item["uuid"], nil
					case "item.name":
						return item["name"], nil
					}
					return expression, nil
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.addCalls)
		assert.Equal(t, 0, memoryCtx.updateCalls)
		assert.Equal(t, 0, memoryCtx.namespaceUpdateCalls)

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 0, metadata["updatedCount"])
		assert.Equal(t, 2, metadata["createdCount"])
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

	t.Run("list mode requires listSource", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":   "deployments",
				"iterateList": true,
				"matchList":   []map[string]any{},
				"valueList": []map[string]any{
					{"name": "value", "value": "abc"},
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listSource")
	})

	t.Run("list mode accepts valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":    "deployments",
				"iterateList":  true,
				"listSource":   "x",
				"itemVariable": "service",
				"matchList":    []map[string]any{},
				"valueList": []map[string]any{
					{"name": "name", "value": "service.name"},
				},
			},
		})
		assert.NoError(t, err)
	})
}
