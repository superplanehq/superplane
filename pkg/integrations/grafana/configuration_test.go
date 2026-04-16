package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__Grafana__timeRangeFieldsUseExpectedInputs(t *testing.T) {
	t.Run("render panel", func(t *testing.T) {
		fields := (&RenderPanel{}).Configuration()
		require.Equal(t, configuration.FieldTypeString, fieldByName(fields, "from").Type)
		require.Equal(t, configuration.FieldTypeString, fieldByName(fields, "to").Type)
		require.Nil(t, fieldByName(fields, "from").Default)
		require.Nil(t, fieldByName(fields, "to").Default)
		require.Equal(t, `{{ now() - duration("1h") }}`, fieldByName(fields, "from").Placeholder)
		require.Equal(t, `{{ now() }}`, fieldByName(fields, "to").Placeholder)
	})

	t.Run("query data source", func(t *testing.T) {
		fields := (&QueryDataSource{}).Configuration()
		require.Equal(t, configuration.FieldTypeString, fieldByName(fields, "timeFrom").Type)
		require.Equal(t, configuration.FieldTypeString, fieldByName(fields, "timeTo").Type)
		require.Nil(t, fieldByName(fields, "timeFrom").Default)
		require.Nil(t, fieldByName(fields, "timeTo").Default)
		require.Equal(t, `{{ now() - duration("5m") }}`, fieldByName(fields, "timeFrom").Placeholder)
		require.Equal(t, `{{ now() }}`, fieldByName(fields, "timeTo").Placeholder)
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
