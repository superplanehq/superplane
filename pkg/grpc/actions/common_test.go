package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	_ "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"google.golang.org/protobuf/proto"
)

type testTrigger struct {
	defaultRunTitle string
}

func (t testTrigger) Name() string { return "test.trigger" }

func (t testTrigger) Label() string { return "Test Trigger" }

func (t testTrigger) Description() string { return "Test trigger" }

func (t testTrigger) Documentation() string { return "" }

func (t testTrigger) Icon() string { return "test" }

func (t testTrigger) Color() string { return "gray" }

func (t testTrigger) ExampleData() map[string]any { return nil }

func (t testTrigger) Configuration() []configuration.Field { return nil }

func (t testTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t testTrigger) Setup(ctx core.TriggerContext) error { return nil }

func (t testTrigger) Actions() []core.Action { return nil }

func (t testTrigger) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t testTrigger) Cleanup(ctx core.TriggerContext) error { return nil }

func (t testTrigger) DefaultRunTitle() string { return t.defaultRunTitle }

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

func TestTriggerDefaultRunTitle(t *testing.T) {
	assert.Equal(
		t,
		"{{ $.data.push.changes[0].new.target.message }}",
		TriggerDefaultRunTitle(testTrigger{
			defaultRunTitle: "{{ $.data.push.changes[0].new.target.message }}",
		}),
	)

	assert.Equal(t, "", TriggerDefaultRunTitle(testTrigger{}))
}

func TestNodeRunTitleTemplateProtoRoundTrip(t *testing.T) {
	runTitleTemplate := "Push {{ $.data.repository.full_name }}"

	nodes := ProtoToNodes([]*componentpb.Node{
		{
			Id:               "node-1",
			Name:             "Node 1",
			Type:             componentpb.Node_TYPE_TRIGGER,
			RunTitleTemplate: proto.String(runTitleTemplate),
			Trigger:          &componentpb.Node_TriggerRef{Name: "bitbucket.onPush"},
		},
	})

	require.Len(t, nodes, 1)
	require.NotNil(t, nodes[0].RunTitleTemplate)
	assert.Equal(t, runTitleTemplate, *nodes[0].RunTitleTemplate)

	pbNodes := NodesToProto(nodes)
	require.Len(t, pbNodes, 1)
	require.NotNil(t, pbNodes[0].RunTitleTemplate)
	assert.Equal(t, runTitleTemplate, pbNodes[0].GetRunTitleTemplate())
}

func TestNodeRunTitleTemplateProtoDropsTriggerDefault(t *testing.T) {
	nodes := ProtoToNodes([]*componentpb.Node{
		{
			Id:               "node-1",
			Name:             "Node 1",
			Type:             componentpb.Node_TYPE_TRIGGER,
			RunTitleTemplate: proto.String("{{ $.data.push.changes[0].new.target.message }}"),
			Trigger:          &componentpb.Node_TriggerRef{Name: "bitbucket.onPush"},
		},
	})

	require.Len(t, nodes, 1)
	assert.Nil(t, nodes[0].RunTitleTemplate)
}
