package setdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

type canvasDataContext struct {
	values map[string]any
}

func (c *canvasDataContext) Set(key string, value any) error {
	c.values[key] = value
	return nil
}

func (c *canvasDataContext) Get(key string) (any, bool, error) {
	value, ok := c.values[key]
	return value, ok, nil
}

func (c *canvasDataContext) List() (map[string]any, error) {
	return c.values, nil
}

func TestSetData_Execute(t *testing.T) {
	component := &SetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{values: map[string]any{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":   "ticket_id",
			"value": "INC-42",
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "INC-42", canvasData.values["ticket_id"])
	assert.Equal(t, "default", stateCtx.Channel)
	assert.Equal(t, PayloadType, stateCtx.Type)
}

func TestSetData_Execute_AppendToList(t *testing.T) {
	component := &SetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"environments": []any{
				map[string]any{
					"pull_request": 122,
					"requester":    "other-user",
				},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":       "environments",
			"operation": "append",
			"value": map[string]any{
				"pull_request": 123,
				"requester":    "shiroyasha",
				"sandbox_id":   "12asdfasdf-12341asd12341-324",
			},
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	list, ok := canvasData.values["environments"].([]any)
	assert.True(t, ok)
	assert.Len(t, list, 2)
}

func TestSetData_Execute_AppendWithUniqueBy(t *testing.T) {
	component := &SetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"environments": []any{
				map[string]any{
					"pull_request": 123,
					"requester":    "old-user",
					"sandbox_id":   "old-sandbox",
				},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":       "environments",
			"operation": "append",
			"uniqueBy":  "pull_request",
			"value": map[string]any{
				"pull_request": 123,
				"requester":    "shiroyasha",
				"sandbox_id":   "12asdfasdf-12341asd12341-324",
			},
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	list, ok := canvasData.values["environments"].([]any)
	assert.True(t, ok)
	assert.Len(t, list, 1)
	updated, ok := list[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "shiroyasha", updated["requester"])
	assert.Equal(t, "12asdfasdf-12341asd12341-324", updated["sandbox_id"])
}

func TestSetData_Execute_BuildObjectFromValueList(t *testing.T) {
	component := &SetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{values: map[string]any{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":       "environment",
			"operation": "set",
			"valueList": []any{
				map[string]any{"name": "pull_request", "value": 123},
				map[string]any{"name": "creator", "value": "shiroyasha"},
				map[string]any{"name": "sandbox_id", "value": "asdfasdf-123asdfasdf-asdfasdf"},
			},
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	objectValue, ok := canvasData.values["environment"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, 123, objectValue["pull_request"])
	assert.Equal(t, "shiroyasha", objectValue["creator"])
	assert.Equal(t, "asdfasdf-123asdfasdf-asdfasdf", objectValue["sandbox_id"])
}

func TestSetData_Execute_WithoutCanvasDataContext(t *testing.T) {
	component := &SetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":   "ticket_id",
			"value": "INC-42",
		},
		ExecutionState: stateCtx,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "canvas data context is not available")
}
