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

func TestNodeConfigurationBuilder_ResolveExpressionWithExtraVariables_BindsIterationVariable(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	out, err := b.ResolveExpressionWithExtraVariables(`item.service`, map[string]any{
		"item": map[string]any{"service": "api"},
	})
	require.NoError(t, err)
	require.Equal(t, "api", out)
}

func TestNodeConfigurationBuilder_ResolveExpressionWithExtraVariables_RejectsReservedNames(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).WithInput(map[string]any{})

	_, err := b.ResolveExpressionWithExtraVariables(`memory`, map[string]any{
		"memory": "hijacked",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved")
}

func TestNodeConfigurationBuilder_ResolveExpression_UsesConfiguredExpressionVariables(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithInput(map[string]any{}).
		WithExpressionVariables(map[string]any{
			"parameters": map[string]any{
				"message": "hello",
			},
		})

	out, err := b.ResolveExpression(`parameters["message"]`)
	require.NoError(t, err)
	require.Equal(t, "hello", out)
}
