package actions

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var _ = registryimports.Loaded

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

	t.Run("roundtrip list type options with Accordion and Reorderable preserves fields", func(t *testing.T) {
		field := configuration.Field{
			Name:  "parameters",
			Label: "Parameters",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:   "Parameter",
					Accordion:   true,
					Reorderable: true,
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
					},
				},
			},
		}

		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.TypeOptions)
		require.NotNil(t, pbField.TypeOptions.List)
		require.NotNil(t, pbField.TypeOptions.List.Accordion)
		require.True(t, *pbField.TypeOptions.List.Accordion)
		require.NotNil(t, pbField.TypeOptions.List.Reorderable)
		require.True(t, *pbField.TypeOptions.List.Reorderable)

		field2 := ProtoToConfigurationField(pbField)
		require.NotNil(t, field2.TypeOptions)
		require.NotNil(t, field2.TypeOptions.List)
		assert.True(t, field2.TypeOptions.List.Accordion)
		assert.True(t, field2.TypeOptions.List.Reorderable)
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

	t.Run("roundtrip integration type options preserves field", func(t *testing.T) {
		field := configuration.Field{
			Name:  "integration",
			Label: "Integration",
			Type:  configuration.FieldTypeIntegration,
			TypeOptions: &configuration.TypeOptions{
				Integration: &configuration.IntegrationTypeOptions{
					Integration: "claude",
				},
			},
		}

		pbField := ConfigurationFieldToProto(field)
		require.NotNil(t, pbField.TypeOptions)
		require.NotNil(t, pbField.TypeOptions.Integration)
		assert.Equal(t, "claude", pbField.TypeOptions.Integration.Integration)

		field2 := ProtoToConfigurationField(pbField)
		require.NotNil(t, field2.TypeOptions)
		require.NotNil(t, field2.TypeOptions.Integration)
		assert.Equal(t, "claude", field2.TypeOptions.Integration.Integration)
	})
}

func TestSerializeTriggersAddsDefaultRunTitleExpression(t *testing.T) {
	triggers := SerializeTriggers([]core.Trigger{
		&testTriggerDefinition{name: "github.onPush"},
	})

	require.Len(t, triggers, 1)
	require.Len(t, triggers[0].Configuration, 1)

	runTitle := triggers[0].Configuration[0]
	require.Equal(t, "customName", runTitle.Name)
	require.Equal(t, "{{ root().data.head_commit.message }} - {{ root().data.head_commit.id[:7] }}", runTitle.GetDefaultValue())
}

func TestDefaultRunTitleExpressionsResolveAgainstExampleData(t *testing.T) {
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	for triggerName, exampleData := range builtInTriggerExamples(reg) {
		t.Run(triggerName, func(t *testing.T) {
			resolved, err := contexts.NewNodeConfigurationBuilder(nil, uuid.Nil).
				WithRootPayload(rootPayloadFromExample(triggerName, exampleData)).
				ResolveTemplateExpressions(defaultRunTitleExpression(triggerName))

			require.NoError(t, err)
			require.NotEmpty(t, resolved)
			require.NotContains(t, resolved, "<nil>")
			require.NotContains(t, resolved, "<no value>")
		})
	}
}

// GitLab sends push events for valid pushes that carry no commits (or omit the
// key entirely). The default title must fall back to the branch ref instead of
// failing to resolve commits[-1].
func TestGitlabOnPushRunTitleFallsBackWithoutCommits(t *testing.T) {
	expression := defaultRunTitleExpression("gitlab.onPush")
	require.NotEmpty(t, expression)

	cases := map[string]map[string]any{
		"empty commits":   {"ref": "refs/heads/main", "commits": []any{}},
		"missing commits": {"ref": "refs/heads/main"},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			resolved, err := contexts.NewNodeConfigurationBuilder(nil, uuid.Nil).
				WithRootPayload(map[string]any{
					"type":      "gitlab.onPush",
					"timestamp": time.Now(),
					"data":      data,
				}).
				ResolveTemplateExpressions(expression)

			require.NoError(t, err)
			require.Equal(t, "refs/heads/main", resolved)
		})
	}
}

func TestBuiltInTriggersHaveDefaultRunTitleExpressions(t *testing.T) {
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	missing := []string{}
	for _, triggerName := range builtInTriggerNames(reg) {
		if defaultRunTitleExpression(triggerName) == "" {
			missing = append(missing, triggerName)
		}
	}

	require.Empty(t, missing, "missing default run title expressions")
}

func builtInTriggerNames(reg *registry.Registry) []string {
	names := []string{}

	for triggerName := range builtInTriggerExamples(reg) {
		names = append(names, triggerName)
	}

	return names
}

func builtInTriggerExamples(reg *registry.Registry) map[string]map[string]any {
	examples := map[string]map[string]any{}

	for _, trigger := range reg.ListTriggers() {
		examples[trigger.Name()] = trigger.ExampleData()
	}

	for _, integration := range reg.ListIntegrations() {
		for _, trigger := range integration.Triggers() {
			examples[trigger.Name()] = trigger.ExampleData()
		}

		setupProvider := reg.SetupProviders[integration.Name()]
		if setupProvider == nil {
			continue
		}

		for _, group := range setupProvider.CapabilityGroups() {
			for _, capability := range group.Capabilities {
				if capability.Type == core.IntegrationCapabilityTypeTrigger {
					if len(capability.ExampleData) > 0 {
						examples[capability.Name] = capability.ExampleData
					} else if _, ok := examples[capability.Name]; !ok {
						examples[capability.Name] = nil
					}
				}
			}
		}
	}

	return examples
}

func rootPayloadFromExample(triggerName string, exampleData map[string]any) map[string]any {
	if exampleData == nil {
		return map[string]any{
			"type":      triggerName,
			"timestamp": time.Now(),
			"data":      map[string]any{},
		}
	}

	if isRootEventExample(exampleData) {
		payload := cloneExampleData(exampleData)
		if _, ok := payload["timestamp"].(time.Time); !ok {
			payload["timestamp"] = time.Now()
		}

		return payload
	}

	return map[string]any{
		"type":      triggerName,
		"timestamp": time.Now(),
		"data":      cloneExampleData(exampleData),
	}
}

func isRootEventExample(exampleData map[string]any) bool {
	_, hasType := exampleData["type"]
	_, hasData := exampleData["data"]
	return hasType && hasData
}

func cloneExampleData(exampleData map[string]any) map[string]any {
	clone := make(map[string]any, len(exampleData))
	for key, value := range exampleData {
		clone[key] = cloneExampleValue(value)
	}

	return clone
}

func cloneExampleValue(value any) any {
	switch typedValue := value.(type) {
	case map[string]any:
		return cloneExampleData(typedValue)
	case []any:
		clone := make([]any, len(typedValue))
		for i, item := range typedValue {
			clone[i] = cloneExampleValue(item)
		}

		return clone
	default:
		return value
	}
}

type testTriggerDefinition struct {
	name string
}

func (t *testTriggerDefinition) Name() string                         { return t.name }
func (t *testTriggerDefinition) Label() string                        { return t.name }
func (t *testTriggerDefinition) Description() string                  { return t.name }
func (t *testTriggerDefinition) Documentation() string                { return "" }
func (t *testTriggerDefinition) Icon() string                         { return "" }
func (t *testTriggerDefinition) Color() string                        { return "" }
func (t *testTriggerDefinition) ExampleData() map[string]any          { return nil }
func (t *testTriggerDefinition) Configuration() []configuration.Field { return nil }
func (t *testTriggerDefinition) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (t *testTriggerDefinition) Setup(core.TriggerContext) error { return nil }
func (t *testTriggerDefinition) Hooks() []core.Hook              { return nil }
func (t *testTriggerDefinition) HandleHook(core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}
func (t *testTriggerDefinition) Cleanup(core.TriggerContext) error { return nil }

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
