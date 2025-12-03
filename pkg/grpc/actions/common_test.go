package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
)

func TestConfigurationFieldToProto(t *testing.T) {
	t.Run("stores default as plain string", func(t *testing.T) {
		defaultValue := "https://example.com/webhook"

		field := configuration.Field{
			Name:    "url",
			Label:   "Webhook URL",
			Type:    configuration.FieldTypeString,
			Default: defaultValue,
		}

		pbField := ConfigurationFieldToProto(field)

		require.NotNil(t, pbField.DefaultValue, "expected DefaultValue to be set")
		assert.Equal(t, defaultValue, *pbField.DefaultValue)
	})

	t.Run("default string does not change during roundtrips", func(t *testing.T) {
		defaultValue := "https://example.com/webhook"

		pbField := &configpb.Field{
			Name:         "url",
			Label:        "Webhook URL",
			Type:         configuration.FieldTypeString,
			DefaultValue: &defaultValue,
		}

		field := ProtoToConfigurationField(pbField)

		got, ok := field.Default.(string)
		require.True(t, ok, "expected Default to be string")
		assert.Equal(t, defaultValue, got)
	})
}
