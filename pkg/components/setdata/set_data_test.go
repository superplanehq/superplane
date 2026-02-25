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
