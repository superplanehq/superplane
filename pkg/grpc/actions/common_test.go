package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
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

func TestCapabilityStateToProto(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected organizationpb.Integration_CapabilityState_State
	}{
		{
			name:     "requested",
			state:    string(core.IntegrationCapabilityStateRequested),
			expected: organizationpb.Integration_CapabilityState_STATE_REQUESTED,
		},
		{
			name:     "enabled",
			state:    string(core.IntegrationCapabilityStateEnabled),
			expected: organizationpb.Integration_CapabilityState_STATE_ENABLED,
		},
		{
			name:     "disabled",
			state:    string(core.IntegrationCapabilityStateDisabled),
			expected: organizationpb.Integration_CapabilityState_STATE_DISABLED,
		},
		{
			name:     "available",
			state:    string(core.IntegrationCapabilityStateAvailable),
			expected: organizationpb.Integration_CapabilityState_STATE_AVAILABLE,
		},
		{
			name:     "unavailable defaults to unavailable",
			state:    string(core.IntegrationCapabilityStateUnavailable),
			expected: organizationpb.Integration_CapabilityState_STATE_UNAVAILABLE,
		},
		{
			name:     "unknown defaults to unavailable",
			state:    "unknown",
			expected: organizationpb.Integration_CapabilityState_STATE_UNAVAILABLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CapabilityStateToProto(tt.state))
		})
	}
}

func TestProtoToCapabilityState(t *testing.T) {
	tests := []struct {
		name     string
		state    organizationpb.Integration_CapabilityState_State
		expected string
	}{
		{
			name:     "available",
			state:    organizationpb.Integration_CapabilityState_STATE_AVAILABLE,
			expected: string(core.IntegrationCapabilityStateAvailable),
		},
		{
			name:     "unavailable",
			state:    organizationpb.Integration_CapabilityState_STATE_UNAVAILABLE,
			expected: string(core.IntegrationCapabilityStateUnavailable),
		},
		{
			name:     "requested",
			state:    organizationpb.Integration_CapabilityState_STATE_REQUESTED,
			expected: string(core.IntegrationCapabilityStateRequested),
		},
		{
			name:     "enabled",
			state:    organizationpb.Integration_CapabilityState_STATE_ENABLED,
			expected: string(core.IntegrationCapabilityStateEnabled),
		},
		{
			name:     "disabled",
			state:    organizationpb.Integration_CapabilityState_STATE_DISABLED,
			expected: string(core.IntegrationCapabilityStateDisabled),
		},
		{
			name:     "unknown returns empty string",
			state:    organizationpb.Integration_CapabilityState_State(99),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ProtoToCapabilityState(tt.state))
		})
	}
}

func TestCapabilityTypeToProto(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		expected integrationpb.CapabilityDefinition_Type
	}{
		{
			name:     "action",
			typ:      string(core.IntegrationCapabilityTypeAction),
			expected: integrationpb.CapabilityDefinition_TYPE_ACTION,
		},
		{
			name:     "trigger",
			typ:      string(core.IntegrationCapabilityTypeTrigger),
			expected: integrationpb.CapabilityDefinition_TYPE_TRIGGER,
		},
		{
			name:     "unknown defaults to unknown",
			typ:      "unknown",
			expected: integrationpb.CapabilityDefinition_TYPE_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CapabilityTypeToProto(tt.typ))
		})
	}
}

func TestProtoToCapabilityType(t *testing.T) {
	tests := []struct {
		name     string
		typ      integrationpb.CapabilityDefinition_Type
		expected string
	}{
		{
			name:     "action",
			typ:      integrationpb.CapabilityDefinition_TYPE_ACTION,
			expected: string(core.IntegrationCapabilityTypeAction),
		},
		{
			name:     "trigger",
			typ:      integrationpb.CapabilityDefinition_TYPE_TRIGGER,
			expected: string(core.IntegrationCapabilityTypeTrigger),
		},
		{
			name:     "unknown returns empty string",
			typ:      integrationpb.CapabilityDefinition_TYPE_UNKNOWN,
			expected: "",
		},
		{
			name:     "unmapped returns empty string",
			typ:      integrationpb.CapabilityDefinition_Type(99),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ProtoToCapabilityType(tt.typ))
		})
	}
}
