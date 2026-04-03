package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Filter__Execute(t *testing.T) {
	filter := &Filter{}

	t.Run("error executing expressions should return error", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}

		ctx := core.ExecutionContext{
			Data:           map[string]any{"test": "value"},
			Configuration:  map[string]any{"expression": "invalid expression syntax +++"},
			ExecutionState: stateCtx,
			Expressions: &contexts.ExpressionContext{
				Error: fmt.Errorf("variable x not found"),
			},
		}

		err := filter.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "variable x not found")
	})

	t.Run("non boolean result should return error", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			ExecutionState: stateCtx,
			Data:           map[string]any{"test": "value"},
			Configuration:  map[string]any{"expression": "$.test"},
			Expressions: &contexts.ExpressionContext{
				Output: "not a boolean",
			},
		}

		err := filter.Execute(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expression must evaluate to boolean, got string: not a boolean")
		assert.False(t, stateCtx.Passed)
	})

	t.Run("true expression emits on default output channel", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			ExecutionState: stateCtx,
			Configuration:  map[string]any{"expression": "true"},
			Expressions: &contexts.ExpressionContext{
				Output: true,
			},
		}

		err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, "filter.executed", stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
	})

	t.Run("false expression finishes successfully, but does not emit", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			ExecutionState: stateCtx,
			Configuration:  map[string]any{"expression": "false"},
			Expressions: &contexts.ExpressionContext{
				Output: false,
			},
		}

		err := filter.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Empty(t, stateCtx.Payloads)
	})
}
