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
	assert.Equal(t, "default", stateCtx.Channel)
	assert.Equal(t, PayloadType, stateCtx.Type)
	assert.Len(t, stateCtx.Payloads, 1)
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
