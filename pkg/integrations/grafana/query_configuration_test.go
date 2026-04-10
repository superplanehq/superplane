package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__GrafanaQueryComponents__Configuration__DoNotExposeTimezoneField(t *testing.T) {
	t.Run("query data source", func(t *testing.T) {
		assertNoTimezoneField(t, (&QueryDataSource{}).Configuration())
	})

	t.Run("query logs", func(t *testing.T) {
		assertNoTimezoneField(t, (&QueryLogs{}).Configuration())
	})

	t.Run("query traces", func(t *testing.T) {
		assertNoTimezoneField(t, (&QueryTraces{}).Configuration())
	})
}

func Test__GrafanaQueryComponents__Configuration__UsesExpressionPlaceholdersForFlexibleTimeRanges(t *testing.T) {
	t.Run("query logs", func(t *testing.T) {
		assertTimeFieldPlaceholder(
			t,
			(&QueryLogs{}).Configuration(),
			"timeFrom",
			`{{ now() - duration("15m") }} or now-15m`,
		)
		assertTimeFieldPlaceholder(
			t,
			(&QueryLogs{}).Configuration(),
			"timeTo",
			`{{ now() + duration("1m") }} or now`,
		)
	})

	t.Run("query traces", func(t *testing.T) {
		assertTimeFieldPlaceholder(
			t,
			(&QueryTraces{}).Configuration(),
			"timeFrom",
			`{{ now() - duration("15m") }} or now-15m`,
		)
		assertTimeFieldPlaceholder(
			t,
			(&QueryTraces{}).Configuration(),
			"timeTo",
			`{{ now() + duration("1m") }} or now`,
		)
	})
}

func assertNoTimezoneField(t *testing.T, fields []configuration.Field) {
	t.Helper()

	for i := range fields {
		field := fields[i]
		require.NotEqual(t, "timezone", field.Name)
	}
}

func assertTimeFieldPlaceholder(t *testing.T, fields []configuration.Field, name, placeholder string) {
	t.Helper()

	for i := range fields {
		field := fields[i]
		if field.Name != name {
			continue
		}

		require.Equal(t, placeholder, field.Placeholder)
		return
	}

	t.Fatalf("field %q not found", name)
}
