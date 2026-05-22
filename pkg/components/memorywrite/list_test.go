package memorywrite

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeExpressionContext struct {
	listOutput any
	listErr    error
	runCalls   []string
	scopeCalls []scopeCall
	scopeFunc  func(expression string, scope map[string]any) (any, error)
}

type scopeCall struct {
	expression string
	scope      map[string]any
}

func (f *fakeExpressionContext) Run(expression string) (any, error) {
	f.runCalls = append(f.runCalls, expression)
	return f.listOutput, f.listErr
}

func (f *fakeExpressionContext) RunWithScope(expression string, scope map[string]any) (any, error) {
	f.scopeCalls = append(f.scopeCalls, scopeCall{expression: expression, scope: scope})
	if f.scopeFunc != nil {
		return f.scopeFunc(expression, scope)
	}
	return nil, nil
}

func TestListMode_NormalizeAppliesDefaultItemVariable(t *testing.T) {
	mode := ListMode{IterateList: true, ListSource: " $x ", ItemVariable: ""}.Normalize()
	assert.Equal(t, "$x", mode.ListSource)
	assert.Equal(t, DefaultItemVariable, mode.ItemVariable)
}

func TestListMode_NormalizeNoOpWhenDisabled(t *testing.T) {
	mode := ListMode{IterateList: false}.Normalize()
	assert.Equal(t, "", mode.ItemVariable)
}

func TestListMode_ValidateRequiresSource(t *testing.T) {
	err := ListMode{IterateList: true, ItemVariable: "item"}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listSource")
}

func TestListMode_ValidateRejectsReservedAndInvalidNames(t *testing.T) {
	cases := []string{"$", "memory", "1bad", "with space", ""}
	for _, name := range cases {
		t.Run(fmt.Sprintf("rejects %q", name), func(t *testing.T) {
			err := ListMode{IterateList: true, ListSource: "list", ItemVariable: name}.Validate()
			require.Error(t, err)
		})
	}
}

func TestListMode_EvaluateListCoercesShapes(t *testing.T) {
	cases := []struct {
		name     string
		output   any
		expected []any
	}{
		{"any slice", []any{1, 2}, []any{1, 2}},
		{"maps slice", []map[string]any{{"a": 1}}, []any{map[string]any{"a": 1}}},
		{"string slice", []string{"x", "y"}, []any{"x", "y"}},
		{"nil", nil, []any{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expressions := &fakeExpressionContext{listOutput: tc.output}
			mode := ListMode{IterateList: true, ListSource: "src", ItemVariable: "item"}
			items, err := mode.EvaluateList(expressions)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, items)
			assert.Equal(t, []string{"src"}, expressions.runCalls)
		})
	}
}

func TestListMode_EvaluateListRejectsNonList(t *testing.T) {
	expressions := &fakeExpressionContext{listOutput: "scalar"}
	mode := ListMode{IterateList: true, ListSource: "src", ItemVariable: "item"}
	_, err := mode.EvaluateList(expressions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listSource must evaluate to a list")
}

func TestResolveValue_StringEvaluatedWithScope(t *testing.T) {
	expressions := &fakeExpressionContext{
		scopeFunc: func(expression string, scope map[string]any) (any, error) {
			assert.Equal(t, "item.service", expression)
			assert.Equal(t, "api", scope["item"].(map[string]any)["service"])
			return "api", nil
		},
	}

	got, err := ResolveValue("item.service", map[string]any{"item": map[string]any{"service": "api"}}, expressions)
	require.NoError(t, err)
	assert.Equal(t, "api", got)
}

func TestResolveValue_NonStringPassThrough(t *testing.T) {
	expressions := &fakeExpressionContext{}
	got, err := ResolveValue(42, map[string]any{"item": "x"}, expressions)
	require.NoError(t, err)
	assert.Equal(t, 42, got)
	assert.Empty(t, expressions.scopeCalls)
}

func TestResolveValue_EmptyStringIsLiteral(t *testing.T) {
	expressions := &fakeExpressionContext{}
	got, err := ResolveValue("   ", map[string]any{}, expressions)
	require.NoError(t, err)
	assert.Equal(t, "   ", got)
	assert.Empty(t, expressions.scopeCalls)
}

func TestResolvePairs_TrimsNamesAndSkipsEmpty(t *testing.T) {
	expressions := &fakeExpressionContext{
		scopeFunc: func(expression string, _ map[string]any) (any, error) {
			if expression == "expr" {
				return 42, nil
			}
			return nil, fmt.Errorf("unexpected %q", expression)
		},
	}

	got, err := ResolvePairs([]NameValuePair{
		{Name: " kept ", Value: "expr"},
		{Name: "literal", Value: 7},
		{Name: "  ", Value: "ignored"},
	}, map[string]any{"item": "x"}, expressions)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"kept": 42, "literal": 7}, got)
}

func TestResolvePairs_WrapsResolveErrors(t *testing.T) {
	expressions := &fakeExpressionContext{
		scopeFunc: func(string, map[string]any) (any, error) {
			return nil, fmt.Errorf("boom")
		},
	}

	_, err := ResolvePairs([]NameValuePair{{Name: "bad", Value: "x"}}, nil, expressions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `field "bad"`)
	assert.Contains(t, err.Error(), "boom")
}
