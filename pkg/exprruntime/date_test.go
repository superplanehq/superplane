package exprruntime

import (
	"testing"
	"time"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"
)

func TestDateFunctionOption_WorksWithExprTimezoneOption(t *testing.T) {
	env := map[string]any{
		"ts": "2026-03-17T01:02:03Z",
	}

	program, err := expr.Compile(
		`date(ts).Add(duration("1ns")).Format("2006-01-02T15:04:05.999999999Z07:00")`,
		expr.Env(env),
		expr.AsAny(),
		expr.Timezone(time.UTC.String()),
		DateFunctionOption(),
	)
	require.NoError(t, err)

	out, err := expr.Run(program, env)
	require.NoError(t, err)
	require.Equal(t, "2026-03-17T01:02:03.000000001Z", out)
}

func TestDateFunctionOption_AcceptsTimezoneAsLocationPointerOrValue(t *testing.T) {
	tzValue := *time.UTC

	for _, tc := range []struct {
		name string
		tz   any
	}{
		{name: "pointer", tz: time.UTC},
		{name: "value", tz: tzValue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]any{
				"tz": tc.tz,
			}

			program, err := expr.Compile(
				`date("2026-03-17", tz).Format("2006-01-02T15:04:05Z07:00")`,
				expr.Env(env),
				expr.AsAny(),
				expr.Timezone(time.UTC.String()),
				DateFunctionOption(),
			)
			require.NoError(t, err)

			out, err := expr.Run(program, env)
			require.NoError(t, err)
			require.Equal(t, "2026-03-17T00:00:00Z", out)
		})
	}
}
