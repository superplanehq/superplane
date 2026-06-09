package loop

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type sequenceExpressionContext struct {
	outputs []any
	index   int
}

func (s *sequenceExpressionContext) Run(_ string) (any, error) {
	if s.index >= len(s.outputs) {
		return nil, fmt.Errorf("unexpected expression at index %d", s.index)
	}
	out := s.outputs[s.index]
	s.index++
	return out, nil
}

func (s *sequenceExpressionContext) RunWithExtraVariables(expression string, variables map[string]any) (any, error) {
	if expression == `{"label": row, "position": index + 1}` {
		return map[string]any{
			"label":    variables["row"],
			"position": variables["index"].(int) + 1,
		}, nil
	}
	return s.Run(expression)
}

func TestLoopSetup(t *testing.T) {
	component := &Loop{}

	t.Run("requires collection expression in collection mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"mode": ModeCollection},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "collectionExpression is required")
	})

	t.Run("requires count expression in count mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"mode": ModeCount},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "countExpression is required")
	})

	t.Run("requires range expressions in range mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"mode": ModeRange},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "startExpression is required")
	})

	t.Run("accepts valid collection configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"mode":                 ModeCollection,
				"collectionExpression": `$["Runner"].items`,
			},
		})
		require.NoError(t, err)
	})
}

func TestLoopExecuteCollectionMode(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	exprCtx := &contexts.ExpressionContext{
		Output: []any{
			map[string]any{"service": "EC2"},
			map[string]any{"service": "S3"},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"mode":                 ModeCollection,
			"collectionExpression": `$["Runner"].items`,
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
		Expressions:    exprCtx,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, ChannelNameIteration, execState.Channel)
	assert.Equal(t, PayloadType, execState.Type)
	require.Len(t, execState.Payloads, 2)

	first := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, map[string]any{"service": "EC2"}, first["item"])
	assert.Equal(t, 0, first["index"])
	assert.Equal(t, 2, first["totalCount"])
	assert.Equal(t, true, first["first"])
	assert.Equal(t, false, first["last"])

	last := execState.Payloads[1].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, true, last["last"])
}

func TestLoopExecuteCountMode(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	exprCtx := &contexts.ExpressionContext{Output: 3}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"mode":            ModeCount,
			"countExpression": `$["Runner"].retries`,
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
		Expressions:    exprCtx,
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 3)

	for i, payload := range execState.Payloads {
		data := payload.(map[string]any)["data"].(map[string]any)
		assert.Equal(t, i, data["item"])
		assert.Equal(t, i, data["index"])
	}
}

func TestLoopExecuteRangeMode(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	exprCtx := &sequenceExpressionContext{outputs: []any{0, 4, 2}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"mode":            ModeRange,
			"startExpression": "0",
			"endExpression":   "4",
			"stepExpression":  "2",
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
		Expressions:    exprCtx,
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 3)

	values := []int{0, 2, 4}
	for i, payload := range execState.Payloads {
		data := payload.(map[string]any)["data"].(map[string]any)
		assert.Equal(t, values[i], data["item"])
	}
}

func TestLoopExecuteCustomItemVariableAndPayload(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	exprCtx := &contexts.ExpressionContext{
		Output: []any{"alpha", "beta"},
		WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
			if expression == `{"label": row, "position": index + 1}` {
				return map[string]any{
					"label":    variables["row"],
					"position": variables["index"].(int) + 1,
				}, nil
			}
			return nil, fmt.Errorf("unexpected expression %q", expression)
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"mode":                 ModeCollection,
			"collectionExpression": `$["Runner"].items`,
			"itemVariable":         "row",
			"payloadExpression":    `{"label": row, "position": index + 1}`,
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
		Expressions:    exprCtx,
	})

	require.NoError(t, err)
	require.Len(t, execState.Payloads, 2)

	first := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "alpha", first["label"])
	assert.Equal(t, 1, first["position"])
}

func TestLoopExecutePassesWhenEmpty(t *testing.T) {
	component := &Loop{}
	execState := &contexts.ExecutionStateContext{}
	execMetadata := &contexts.MetadataContext{}
	exprCtx := &contexts.ExpressionContext{Output: []any{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"mode":                 ModeCollection,
			"collectionExpression": `$["Runner"].items`,
		},
		Metadata:       execMetadata,
		ExecutionState: execState,
		Expressions:    exprCtx,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Empty(t, execState.Payloads)
}

func TestLoopExecuteErrors(t *testing.T) {
	component := &Loop{}

	t.Run("collection must evaluate to a list", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"mode":                 ModeCollection,
				"collectionExpression": `$["Runner"].name`,
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Expressions:    &contexts.ExpressionContext{Output: "not-a-list"},
		})
		assert.ErrorContains(t, err, "collection expression must evaluate to a list")
	})

	t.Run("rejects iteration count above limit", func(t *testing.T) {
		items := make([]any, core.MaxEmitCount+1)
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"mode":                 ModeCollection,
				"collectionExpression": `$["Runner"].items`,
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Expressions:    &contexts.ExpressionContext{Output: items},
		})
		assert.ErrorContains(t, err, "supports at most")
	})

	t.Run("rejects reserved item variable", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"mode":                 ModeCollection,
				"collectionExpression": `$["Runner"].items`,
				"itemVariable":         "index",
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Expressions:    &contexts.ExpressionContext{Output: []any{"x"}},
		})
		assert.ErrorContains(t, err, "itemVariable \"index\" is reserved")
	})
}

func TestBuildRangeValues(t *testing.T) {
	t.Run("inclusive positive step", func(t *testing.T) {
		values, err := buildRangeValues(0, 4, 2)
		require.NoError(t, err)
		assert.Equal(t, []any{0, 2, 4}, values)
	})

	t.Run("empty when start is after end with positive step", func(t *testing.T) {
		values, err := buildRangeValues(5, 1, 1)
		require.NoError(t, err)
		assert.Empty(t, values)
	})

	t.Run("supports negative step", func(t *testing.T) {
		values, err := buildRangeValues(3, 0, -1)
		require.NoError(t, err)
		assert.Equal(t, []any{3, 2, 1, 0}, values)
	})
}

func TestCoerceToList(t *testing.T) {
	items, err := coerceToList([]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b"}, items)
}
