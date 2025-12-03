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
}
