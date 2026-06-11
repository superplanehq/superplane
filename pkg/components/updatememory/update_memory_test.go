package updatememory

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
	namespace      string
	matches        map[string]any
	values         map[string]any
	valuesPerCall  []map[string]any
	matchesPerCall []map[string]any
	updatedValues  []any
	updatedPerCall [][]any
	recordsPerCall [][]core.CanvasMemoryRecord
	updateCalls    int
	err            error
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
	c.valuesPerCall = append(c.valuesPerCall, values)
	c.matchesPerCall = append(c.matchesPerCall, matches)
	if c.err != nil {
		return nil, c.err
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
	c.valuesPerCall = append(c.valuesPerCall, values)
	c.matchesPerCall = append(c.matchesPerCall, matches)
	if c.err != nil {
		return nil, c.err
	}
	if len(c.recordsPerCall) == 0 {
		return nil, nil
	}

	idx := c.updateCalls - 1
	if idx >= len(c.recordsPerCall) {
		idx = len(c.recordsPerCall) - 1
	}
	return c.recordsPerCall[idx], nil
}

func memoryRecord(id string, values any) core.CanvasMemoryRecord {
	return core.CanvasMemoryRecord{
		ID:     uuid.MustParse(id),
		Values: values,
	}
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
		execMetadata := &contexts.MetadataContext{}

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
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
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
			execMetadata.Get(),
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

	t.Run("list mode updates once per element with same matches", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{
			recordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"name": "api", "status": "running"})},
				{memoryRecord("22222222-2222-2222-2222-222222222222", map[string]any{"name": "worker", "status": "running"})},
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
				item := variables["item"].(map[string]any)
				switch expression {
				case "item.name":
					return item["name"], nil
				case "prod":
					return "prod", nil
				}
				return "running", nil
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   `$["Runner"].data.services`,
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "env", "value": "prod"},
				},
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
					{"name": "status", "value": "\"running\""},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions:    expressions,
		})

		require.NoError(t, err)
		require.Equal(t, 2, memoryCtx.updateCalls)
		assert.Equal(t, ChannelNameFound, execState.Channel)
		for _, matches := range memoryCtx.matchesPerCall {
			assert.Equal(t, map[string]any{"env": "prod"}, matches)
		}
		assert.Equal(t, "api", memoryCtx.valuesPerCall[0]["name"])
		assert.Equal(t, "worker", memoryCtx.valuesPerCall[1]["name"])

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, metadata["iterateList"])
		assert.Equal(t, "item", metadata["itemVariable"])
		assert.Equal(t, 2, metadata["count"])
		assert.Equal(t, 2, metadata["updatedCount"])
	})

	t.Run("list mode deduplicates repeated updated record", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{
			recordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"name": "api", "status": "starting"})},
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"name": "api", "status": "running"})},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "env", "value": "prod"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "item.status"},
				},
			},
			Metadata:       execMetadata,
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: []any{
					map[string]any{"status": "starting"},
					map[string]any{"status": "running"},
				},
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					return variables["item"].(map[string]any)["status"], nil
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 2, memoryCtx.updateCalls)
		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, metadata["updatedCount"])

		require.Len(t, execState.Payloads, 1)
		payload, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		outerData, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		innerData, ok := outerData["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 1, innerData["count"])
		assert.Equal(t, []any{
			map[string]any{"name": "api", "status": "running"},
		}, innerData["updated"])
	})

	t.Run("list mode resolves matchList per item with iteration variable", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		memoryCtx := &canvasMemoryContext{
			recordsPerCall: [][]core.CanvasMemoryRecord{
				{memoryRecord("11111111-1111-1111-1111-111111111111", map[string]any{"uuid": "A", "status": "running"})},
				{memoryRecord("22222222-2222-2222-2222-222222222222", map[string]any{"uuid": "B", "status": "stopped"})},
			},
		}

		items := []any{
			map[string]any{"uuid": "A", "status": "running"},
			map[string]any{"uuid": "B", "status": "stopped"},
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
					{"name": "status", "value": "item.status"},
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
					case "item.status":
						return item["status"], nil
					}
					return expression, nil
				},
			},
		})

		require.NoError(t, err)
		require.Equal(t, 2, memoryCtx.updateCalls)
		assert.Equal(t, ChannelNameFound, execState.Channel)

		require.Len(t, memoryCtx.matchesPerCall, 2)
		assert.Equal(t, map[string]any{"uuid": "A"}, memoryCtx.matchesPerCall[0])
		assert.Equal(t, map[string]any{"uuid": "B"}, memoryCtx.matchesPerCall[1])

		metadata, ok := execMetadata.Get().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, metadata["updatedCount"])

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
	})

	t.Run("list mode emits notFound when nothing matches", func(t *testing.T) {
		component := &UpdateMemory{}
		execState := &contexts.ExecutionStateContext{}
		memoryCtx := &canvasMemoryContext{updatedValues: []any{}}

		items := []any{map[string]any{"name": "api"}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"namespace":    "services",
				"iterateList":  true,
				"listSource":   "list",
				"itemVariable": "item",
				"matchList": []map[string]any{
					{"name": "env", "value": "prod"},
				},
				"valueList": []map[string]any{
					{"name": "name", "value": "item.name"},
				},
			},
			Metadata:       &contexts.MetadataContext{},
			NodeMetadata:   &contexts.MetadataContext{},
			CanvasMemory:   memoryCtx,
			ExecutionState: execState,
			Expressions: &contexts.ExpressionContext{
				Output: items,
				WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
					return variables["item"].(map[string]any)["name"], nil
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, ChannelNameNotFound, execState.Channel)
		assert.Equal(t, 1, memoryCtx.updateCalls)
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

	t.Run("list mode requires listSource", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":   "machines",
				"iterateList": true,
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "running"},
				},
			},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listSource")
	})

	t.Run("list mode accepts valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"namespace":    "machines",
				"iterateList":  true,
				"listSource":   "x",
				"itemVariable": "service",
				"matchList": []map[string]any{
					{"name": "creator", "value": "igor"},
				},
				"valueList": []map[string]any{
					{"name": "status", "value": "service.status"},
				},
			},
		})
		assert.NoError(t, err)
	})
}
