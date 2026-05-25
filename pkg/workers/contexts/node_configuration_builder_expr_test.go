package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNodeConfigurationBuilder_ResolveExpression_DateWithTimezoneOption(t *testing.T) {
	// This is a regression test for expr runtime crashes when compiling with expr.Timezone("UTC")
	// and using date(...) in server-side expression resolution.
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	out, err := b.ResolveTemplateExpressions(`{{ date("2026-03-17T01:02:03Z").Add(duration("1ns")).Format("2006-01-02T15:04:05.999999999Z07:00") }}`)
	require.NoError(t, err)
	require.Equal(t, "2026-03-17T01:02:03.000000001Z", out)
}

// Regression tests for issue #4499: expressions in JSON body fields resolve
// to strings, breaking numeric APIs. When the entire field value is a single
// {{ ... }} expression, the result's type must be preserved so that numbers,
// booleans, and objects flow through to json.Marshal correctly.
func TestNodeConfigurationBuilder_ResolveTemplateExpressions_PreservesType(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	t.Run("single float expression preserves float64", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ 0.9 }}`)
		require.NoError(t, err)
		require.Equal(t, 0.9, out)
	})

	t.Run("single conditional expression preserves float64", func(t *testing.T) {
		// Mirrors the issue's Cloudflare LB pool weights example.
		out, err := b.ResolveTemplateExpressions(`{{ true ? 0.9 : 0.1 }}`)
		require.NoError(t, err)
		require.Equal(t, 0.9, out)
	})

	t.Run("single int expression preserves int", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ 42 }}`)
		require.NoError(t, err)
		require.Equal(t, 42, out)
	})

	t.Run("single bool expression preserves bool", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ 1 == 1 }}`)
		require.NoError(t, err)
		require.Equal(t, true, out)
	})

	t.Run("single map expression preserves map", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ {"x": 1, "y": 2} }}`)
		require.NoError(t, err)
		require.Equal(t, map[string]any{"x": 1, "y": 2}, out)
	})

	t.Run("single string expression still returns string", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ "hello" }}`)
		require.NoError(t, err)
		require.Equal(t, "hello", out)
	})

	t.Run("mixed text with expression returns string", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`prefix-{{ 0.9 }}`)
		require.NoError(t, err)
		require.Equal(t, "prefix-0.9", out)
	})

	t.Run("multiple expressions return string", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ 1 }}{{ 2 }}`)
		require.NoError(t, err)
		require.Equal(t, "12", out)
	})

	t.Run("plain text without expression returns string", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`https://api.example.com`)
		require.NoError(t, err)
		require.Equal(t, "https://api.example.com", out)
	})

	t.Run("surrounding whitespace prevents type preservation", func(t *testing.T) {
		// Conservative behavior: only the exact single-expression form preserves
		// type. Anything outside the braces — including whitespace — falls back
		// to the string-concat path. Keeps the rule simple and predictable.
		out, err := b.ResolveTemplateExpressions(` {{ 0.9 }} `)
		require.NoError(t, err)
		require.Equal(t, " 0.9 ", out)
	})
}
