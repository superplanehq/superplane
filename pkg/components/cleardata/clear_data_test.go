package cleardata

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

func TestClearData_OutputChannels(t *testing.T) {
	component := &ClearData{}
	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 2)
	assert.Equal(t, "cleared", channels[0].Name)
	assert.Equal(t, "notFound", channels[1].Name)
}

func TestClearData_Execute_RemovesMatchingItems(t *testing.T) {
	component := &ClearData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"pr_sandboxes": []any{
				map[string]any{"pull_request": 15, "sandbox_id": "sb-1"},
				map[string]any{"pull_request": 16, "sandbox_id": "sb-2"},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":        "pr_sandboxes",
			"matchBy":    "pull_request",
			"matchValue": 15,
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "cleared", stateCtx.Channel)
	assert.Equal(t, PayloadType, stateCtx.Type)

	list, ok := canvasData.values["pr_sandboxes"].([]any)
	assert.True(t, ok)
	assert.Len(t, list, 1)

	entry, ok := list[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, 16, entry["pull_request"])
}

func TestClearData_Execute_NoMatch(t *testing.T) {
	component := &ClearData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"pr_sandboxes": []any{
				map[string]any{"pull_request": 16, "sandbox_id": "sb-2"},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":        "pr_sandboxes",
			"matchBy":    "pull_request",
			"matchValue": 15,
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "notFound", stateCtx.Channel)
}

func TestClearData_Execute_KeyMissing(t *testing.T) {
	component := &ClearData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{values: map[string]any{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":        "pr_sandboxes",
			"matchBy":    "pull_request",
			"matchValue": 15,
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "notFound", stateCtx.Channel)
}

func TestClearData_Execute_KeyIsNotList(t *testing.T) {
	component := &ClearData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"pr_sandboxes": map[string]any{"pull_request": 15},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":        "pr_sandboxes",
			"matchBy":    "pull_request",
			"matchValue": 15,
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a list")
}
