package grafana

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testExpressionContext struct {
	run func(string) (any, error)
}

func (t testExpressionContext) Run(expression string) (any, error) {
	return t.run(expression)
}

func Test__resolveGrafanaTimeInput(t *testing.T) {
	t.Run("preserves grafana relative values", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput("now-24h", nil, nil)
		require.NoError(t, err)
		require.Equal(t, "now-24h", value)
	})

	t.Run("evaluates bare expressions to unix millis", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput(
			`now() - duration("24h")`,
			nil,
			testExpressionContext{
				run: func(expression string) (any, error) {
					require.Equal(t, `now() - duration("24h")`, expression)
					return time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC), nil
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "1775721600000", value)
	})

	t.Run("evaluates wrapped template expressions to unix millis", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput(
			`{{ now() - duration("24h") }}`,
			nil,
			testExpressionContext{
				run: func(expression string) (any, error) {
					require.Equal(t, `now() - duration("24h")`, expression)
					return time.Date(2026, 4, 9, 8, 0, 0, 0, time.UTC), nil
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "1775721600000", value)
	})

	t.Run("accepts resolved go time strings with monotonic suffix", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput(
			"2026-04-09 11:27:28.4432 +0300 EAT m=+0.123456789",
			nil,
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, "1775723248443", value)
	})

	t.Run("returns empty value for nil expression result", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput(
			`someExpression()`,
			nil,
			testExpressionContext{
				run: func(expression string) (any, error) {
					require.Equal(t, `someExpression()`, expression)
					return nil, nil
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "", value)
	})

	t.Run("evaluates expressions that return relative strings", func(t *testing.T) {
		value, err := resolveGrafanaTimeInput(
			`someExpression()`,
			nil,
			testExpressionContext{
				run: func(expression string) (any, error) {
					require.Equal(t, `someExpression()`, expression)
					return "now+1h", nil
				},
			},
		)
		require.NoError(t, err)
		require.Equal(t, "now+1h", value)
	})
}
