package contexts

import (
	"encoding/json"
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

func TestNodeConfigurationBuilder_ResolveTemplateExpressions_FormatsLargeFloatIDsWithoutScientificNotation(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithRootPayload(map[string]any{
			"dropletId": float64(566712522),
		}).
		WithInput(map[string]any{
			"source": map[string]any{
				"data": map[string]any{
					"id": float64(566712522),
				},
			},
		})

	t.Run("root payload large integer", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ root().dropletId }}`)
		require.NoError(t, err)
		require.Equal(t, "566712522", out)
	})

	t.Run("previous payload large integer", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ previous().data.id }}`)
		require.NoError(t, err)
		require.Equal(t, "566712522", out)
	})

	t.Run("embedded URL", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`https://api.digitalocean.com/v2/droplets/{{ root().dropletId }}`)
		require.NoError(t, err)
		require.Equal(t, "https://api.digitalocean.com/v2/droplets/566712522", out)
	})
}

func TestNodeConfigurationBuilder_ResolveTemplateExpressions_FormatsNonIntegerNumbers(t *testing.T) {
	b := NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithRootPayload(map[string]any{
			"weight": float64(0.1),
			"rawID":  json.Number("566712522"),
		})

	t.Run("fractional float", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ root().weight }}`)
		require.NoError(t, err)
		require.Equal(t, "0.1", out)
	})

	t.Run("json number", func(t *testing.T) {
		out, err := b.ResolveTemplateExpressions(`{{ root().rawID }}`)
		require.NoError(t, err)
		require.Equal(t, "566712522", out)
	})
}
