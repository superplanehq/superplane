package factories

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/protobuf/types/known/structpb"
)

const customAutomationIntegrationMode = models.CustomAutomationIntegrationStartMode

func parseCustomAutomationIntegrationTrigger(input *pb.FactoryCustomAutomation_IntegrationTrigger) (*pb.FactoryCustomAutomation_IntegrationTrigger, string, error) {
	integrationID, err := uuid.Parse(strings.TrimSpace(input.GetIntegrationId()))
	if err != nil {
		return nil, "", invalidArgument("Select an integration for each integration trigger.")
	}
	trigger := strings.TrimSpace(input.GetTrigger())
	if !strings.Contains(trigger, ".") {
		return nil, "", invalidArgument("Select an event for each integration trigger.")
	}
	configuration := input.GetConfiguration()
	if configuration == nil {
		configuration = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	parsed := &pb.FactoryCustomAutomation_IntegrationTrigger{
		IntegrationId: integrationID.String(),
		Trigger:       trigger,
		Configuration: configuration,
	}
	return parsed, customAutomationIntegrationKey(parsed), nil
}

// The node ID follows the trigger settings. A changed setting publishes a new
// node, and the canvas sets up its subscription again.
func customAutomationIntegrationKey(trigger *pb.FactoryCustomAutomation_IntegrationTrigger) string {
	configuration, _ := json.Marshal(trigger.GetConfiguration().AsMap())
	sum := sha256.Sum256([]byte(trigger.GetIntegrationId() + "|" + trigger.GetTrigger() + "|" + string(configuration)))
	return hex.EncodeToString(sum[:])[:16]
}

func customAutomationIntegrationNode(provider string, settings *customAutomationSettings, trigger customAutomationTrigger, y int) yaml.Node {
	return yaml.Node{
		ID:            "integration-" + customAutomationIntegrationKey(trigger.Integration),
		Name:          "Integration event",
		Type:          yaml.NodeTypeTrigger,
		Component:     trigger.Integration.GetTrigger(),
		Configuration: trigger.Integration.GetConfiguration().AsMap(),
		Integration:   &yaml.IntegrationRef{ID: trigger.Integration.GetIntegrationId()},
		Metadata: customAutomationNodeMetadata(provider, settings.Icon, settings.RunsOnTask, func(template map[string]any) {
			template["startMode"] = customAutomationIntegrationMode
			template["triggerIndex"] = trigger.Index
		}),
		Position: yaml.Position{X: 120, Y: y},
	}
}

func isCustomAutomationIntegrationNode(node models.Node) bool {
	if node.Type != models.NodeTypeTrigger {
		return false
	}
	template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	return configString(template, "startMode") == customAutomationIntegrationMode
}

// Webhook and integration triggers start runs with no task.
func isCustomAutomationEventTriggerNode(node models.Node) bool {
	return node.ComponentName() == factorycomponent.OnWebhookTriggerName || isCustomAutomationIntegrationNode(node)
}

func decodeCustomAutomationIntegrationTrigger(node models.Node) (*pb.FactoryCustomAutomation_IntegrationTrigger, error) {
	configuration, err := structpb.NewStruct(node.Configuration)
	if err != nil {
		return nil, err
	}
	trigger := &pb.FactoryCustomAutomation_IntegrationTrigger{
		Trigger:       node.ComponentName(),
		Configuration: configuration,
	}
	if node.IntegrationID != nil {
		trigger.IntegrationId = *node.IntegrationID
	}
	return trigger, nil
}

// Canvas parsing records trigger setup errors on the node. A custom
// automation must not save a trigger that cannot receive events.
func checkCustomAutomationIntegrationNodes(nodes []models.Node) error {
	for _, node := range nodes {
		if !isCustomAutomationIntegrationNode(node) || node.ErrorMessage == nil {
			continue
		}
		return invalidArgument("Check the integration trigger settings: " + *node.ErrorMessage)
	}
	return nil
}
