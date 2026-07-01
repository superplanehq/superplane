package foreach

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestForEachExecute(t *testing.T) {
	t.Run("emits one payload per array item", func(t *testing.T) {
		component := &ForEach{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		exprCtx := &contexts.ExpressionContext{
			Output: []any{
				map[string]any{"service": "EC2", "cost": 10.0},
				map[string]any{"service": "S3", "cost": 5.0},
				map[string]any{"service": "RDS", "cost": 20.0},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"arrayExpression": `$["Runner"].by_service`},
			Metadata:       execMetadata,
			ExecutionState: execState,
			Expressions:    exprCtx,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, ChannelNameItem, execState.Channel)
		assert.Equal(t, PayloadType, execState.Type)
		require.Len(t, execState.Payloads, 3)

		first := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, map[string]any{"service": "EC2", "cost": 10.0}, first["item"])
		assert.Equal(t, 0, first["index"])
		assert.Equal(t, 3, first["totalCount"])

		last := execState.Payloads[2].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, map[string]any{"service": "RDS", "cost": 20.0}, last["item"])
		assert.Equal(t, 2, last["index"])
	})

	t.Run("passes when array is empty", func(t *testing.T) {
		component := &ForEach{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		exprCtx := &contexts.ExpressionContext{Output: []any{}}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"arrayExpression": `$["Runner"].items`},
			Metadata:       execMetadata,
			ExecutionState: execState,
			Expressions:    exprCtx,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "", execState.Channel)
		assert.Empty(t, execState.Payloads)
	})

	t.Run("returns error when expression result is not an array", func(t *testing.T) {
		component := &ForEach{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		exprCtx := &contexts.ExpressionContext{Output: "not-an-array"}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"arrayExpression": `$["Runner"].name`},
			Metadata:       execMetadata,
			ExecutionState: execState,
			Expressions:    exprCtx,
		})

		assert.ErrorContains(t, err, "expression must evaluate to an array")
	})

	t.Run("returns error when array exceeds item limit", func(t *testing.T) {
		t.Setenv("SUPERPLANE_MAX_EMIT_COUNT", "")

		component := &ForEach{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		items := make([]any, config.MaxEmitCount()+1)
		exprCtx := &contexts.ExpressionContext{Output: items}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"arrayExpression": `$["Runner"].items`},
			Metadata:       execMetadata,
			ExecutionState: execState,
			Expressions:    exprCtx,
		})

		assert.ErrorContains(t, err, "supports at most 100 items per execution")
	})

	t.Run("returns error when arrayExpression is missing", func(t *testing.T) {
		component := &ForEach{}
		execState := &contexts.ExecutionStateContext{}
		execMetadata := &contexts.MetadataContext{}
		exprCtx := &contexts.ExpressionContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			Metadata:       execMetadata,
			ExecutionState: execState,
			Expressions:    exprCtx,
		})

		assert.ErrorContains(t, err, "arrayExpression is required")
	})
}
