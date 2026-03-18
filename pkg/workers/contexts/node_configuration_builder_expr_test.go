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

	out, err := b.ResolveExpression(`{{ date("2026-03-17T01:02:03Z").Add(duration("1ns")).Format("2006-01-02T15:04:05.999999999Z07:00") }}`)
	require.NoError(t, err)
	require.Equal(t, "2026-03-17T01:02:03.000000001Z", out)
}
