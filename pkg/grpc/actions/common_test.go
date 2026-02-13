package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestConfigurationFieldToProto(t *testing.T) {
	t.Run("roundtrip string default value does not introduce extra quotes", func(t *testing.T) {
		original := "https://example.com/webhook"

		field := configuration.Field{
			Name:    "url",
			Label:   "Webhook URL",
			Type:    configuration.FieldTypeString,
			Default: original,
		}

		// First roundtrip
		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.DefaultValue, "expected DefaultValue to be set")

		field2 := ProtoToConfigurationField(pbField)
		got1, ok := field2.Default.(string)
		require.True(t, ok, "expected Default to be string after first roundtrip")
		assert.Equal(t, original, got1)

		// Second roundtrip to ensure we don't accumulate quotes
		pbField2 := ConfigurationFieldToProto(field2)
		require.NotNil(t, pbField2.DefaultValue, "expected DefaultValue to be set on second roundtrip")

		field3 := ProtoToConfigurationField(pbField2)
		got2, ok := field3.Default.(string)
		require.True(t, ok, "expected Default to be string after second roundtrip")
		assert.Equal(t, original, got2)
	})

	t.Run("roundtrip non-string default value works correctly", func(t *testing.T) {
		original := []string{"monday", "wednesday"}

		field := configuration.Field{
			Name:    "days",
			Label:   "Days",
			Type:    configuration.FieldTypeList,
			Default: original,
		}

		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.DefaultValue, "expected DefaultValue to be set")

		field2 := ProtoToConfigurationField(pbField)

		got, ok := field2.Default.([]any)
		require.True(t, ok, "expected Default to be slice after roundtrip")
		require.Len(t, got, len(original))

		for i, v := range got {
			assert.Equal(t, original[i], v)
		}
	})

	t.Run("roundtrip list type options with MaxItems preserves field", func(t *testing.T) {
		maxItems := 4

		field := configuration.Field{
			Name:  "buttons",
			Label: "Buttons",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Button",
					MaxItems:  &maxItems,
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		}

		// Convert to proto
		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.TypeOptions, "expected TypeOptions to be set")
		require.NotNil(t, pbField.TypeOptions.List, "expected List options to be set")
		require.NotNil(t, pbField.TypeOptions.List.MaxItems, "expected MaxItems to be set in proto")
		assert.Equal(t, int32(maxItems), *pbField.TypeOptions.List.MaxItems)

		// Convert back from proto
		field2 := ProtoToConfigurationField(pbField)
		require.NotNil(t, field2.TypeOptions, "expected TypeOptions to be set after roundtrip")
		require.NotNil(t, field2.TypeOptions.List, "expected List options to be set after roundtrip")
		require.NotNil(t, field2.TypeOptions.List.MaxItems, "expected MaxItems to be set after roundtrip")
		assert.Equal(t, maxItems, *field2.TypeOptions.List.MaxItems)
	})
}
