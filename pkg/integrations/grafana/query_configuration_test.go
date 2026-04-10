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

func assertNoTimezoneField(t *testing.T, fields []configuration.Field) {
	t.Helper()

	for i := range fields {
		field := fields[i]
		require.NotEqual(t, "timezone", field.Name)
	}
}
