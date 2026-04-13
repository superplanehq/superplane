package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__Grafana__timeRangeFieldsUseExpectedInputs(t *testing.T) {
	t.Run("render panel", func(t *testing.T) {
		fields := (&RenderPanel{}).Configuration()
		require.Equal(t, configuration.FieldTypeExpression, fieldByName(fields, "from").Type)
		require.Equal(t, configuration.FieldTypeExpression, fieldByName(fields, "to").Type)
	})

	t.Run("query data source", func(t *testing.T) {
		fields := (&QueryDataSource{}).Configuration()
		require.Equal(t, configuration.FieldTypeExpression, fieldByName(fields, "timeFrom").Type)
		require.Equal(t, configuration.FieldTypeExpression, fieldByName(fields, "timeTo").Type)
	})
}

func fieldByName(fields []configuration.Field, name string) configuration.Field {
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}

	return configuration.Field{}
}
