package display

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Display__Execute(t *testing.T) {
	component := &Display{}

	t.Run("stores resolved value and default color, then emits passthrough payload", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"existing": "value",
			},
		}
		input := map[string]any{
			"source": map[string]any{
				"message": "ok",
			},
		}

		ctx := core.ExecutionContext{
			Data:           input,
			Configuration:  map[string]any{"value": "Build completed"},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
		}

		err := component.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
		assert.Len(t, stateCtx.Payloads, 1)
		payload := stateCtx.Payloads[0].(map[string]any)
		assert.Equal(t, input, payload["data"])

		metadata := metadataCtx.Metadata.(map[string]any)
		assert.Equal(t, "value", metadata["existing"])
		assert.Equal(
			t,
			map[string]any{
				"value": "Build completed",
				"color": DefaultColor,
			},
			metadata["display_result"],
		)
	})

	t.Run("resolves expression templates for value and color", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"value": "{{ $['Node'].data.message }}",
				"color": "{{ $['Node'].data.color }}",
			},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
			Expressions: &displayTestExpressionContext{
				outputs: map[string]any{
					"$['Node'].data.message": "Release Ready",
					"$['Node'].data.color":   "green",
				},
			},
		}

		err := component.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
		metadata := metadataCtx.Metadata.(map[string]any)
		assert.Equal(
			t,
			map[string]any{
				"value": "Release Ready",
				"color": "green",
			},
			metadata["display_result"],
		)
	})

	t.Run("expression errors never fail execution and store fallback result", func(t *testing.T) {
		stateCtx := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"value": "{{ $['missing'].data.value }}",
				"color": "{{ $['missing'].data.color }}",
			},
			ExecutionState: stateCtx,
			Metadata:       metadataCtx,
			Expressions: &contexts.ExpressionContext{
				Error: fmt.Errorf("node missing not found"),
			},
		}

		err := component.Execute(ctx)
		assert.NoError(t, err)
		assert.True(t, stateCtx.Passed)
		assert.True(t, stateCtx.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, PayloadType, stateCtx.Type)
		metadata := metadataCtx.Metadata.(map[string]any)
		assert.Equal(
			t,
			map[string]any{
				"value": "[expression error: node missing not found]",
				"color": DefaultColor,
			},
			metadata["display_result"],
		)
	})
}

type displayTestExpressionContext struct {
	outputs map[string]any
}

func (c *displayTestExpressionContext) Run(expression string) (any, error) {
	if output, ok := c.outputs[expression]; ok {
		return output, nil
	}

	return nil, fmt.Errorf("unexpected expression: %s", expression)
}
