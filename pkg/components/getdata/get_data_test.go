package getdata

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

func TestGetData_Execute(t *testing.T) {
	component := &GetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{values: map[string]any{"ticket_id": "INC-42"}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key": "ticket_id",
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "found", stateCtx.Channel)
	assert.Equal(t, PayloadType, stateCtx.Type)
	assert.Len(t, stateCtx.Payloads, 1)
}

func TestGetData_OutputChannels(t *testing.T) {
	component := &GetData{}
	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 2)
	assert.Equal(t, "found", channels[0].Name)
	assert.Equal(t, "notFound", channels[1].Name)
}

func TestGetData_Execute_ListLookupWithReturnField(t *testing.T) {
	component := &GetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"ephemeral_environments": []any{
				map[string]any{"pull_request": 1, "sandbox_id": "adfasdf"},
				map[string]any{"pull_request": 2, "sandbox_id": "sfasdfads"},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":         "ephemeral_environments",
			"mode":        "listLookup",
			"matchBy":     "pull_request",
			"matchValue":  1,
			"returnField": "sandbox_id",
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "found", stateCtx.Channel)
	assert.Len(t, stateCtx.Payloads, 1)
	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, true, data["exists"])
	assert.Equal(t, "adfasdf", data["value"])
}

func TestGetData_Execute_ListLookup_NotFound(t *testing.T) {
	component := &GetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}
	canvasData := &canvasDataContext{
		values: map[string]any{
			"ephemeral_environments": []any{
				map[string]any{"pull_request": 2, "sandbox_id": "sfasdfads"},
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key":        "ephemeral_environments",
			"mode":       "listLookup",
			"matchBy":    "pull_request",
			"matchValue": 1,
		},
		CanvasData:     canvasData,
		ExecutionState: stateCtx,
	})

	assert.NoError(t, err)
	assert.Equal(t, "notFound", stateCtx.Channel)
	assert.Len(t, stateCtx.Payloads, 1)
	payload, ok := stateCtx.Payloads[0].(map[string]any)
	assert.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, false, data["exists"])
	assert.Nil(t, data["value"])
}

func TestGetData_Execute_WithoutCanvasDataContext(t *testing.T) {
	component := &GetData{}
	stateCtx := &supportcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"key": "ticket_id",
		},
		ExecutionState: stateCtx,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "canvas data context is not available")
}
