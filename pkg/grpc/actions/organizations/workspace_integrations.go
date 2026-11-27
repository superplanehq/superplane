package organizations

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type workspaceIntegrationUsage struct {
	integrationIDs map[uuid.UUID]bool
	webhookIDs     map[string]bool
}

func integrationUsageForWorkspace(tx *gorm.DB, registry *registry.Registry, factory *models.Factory, integrations []models.Integration) (*workspaceIntegrationUsage, error) {
	usage := &workspaceIntegrationUsage{integrationIDs: map[uuid.UUID]bool{}, webhookIDs: map[string]bool{}}
	config := factory.OnboardingConfigValue()
	for _, ref := range []string{config.VCSIntegrationID, config.AgentIntegrationID} {
		if id, err := uuid.Parse(ref); err == nil {
			usage.integrationIDs[id] = true
		}
	}
	nodes, err := factory.ListIntegrationNodes(tx)
	if err != nil {
		return nil, err
	}
	byName := map[string]uuid.UUID{}
	for _, integration := range integrations {
		byName[integration.InstallationName] = integration.ID
	}
	for _, node := range nodes {
		if node.AppInstallationID != nil {
			usage.integrationIDs[*node.AppInstallationID] = true
		}
		if node.WebhookID != nil {
			usage.webhookIDs[node.WebhookID.String()] = true
		}
		var fields []configuration.Field
		if action, err := registry.GetAction(node.ComponentName()); err == nil {
			fields = action.Configuration()
		}
		if node.Type == models.NodeTypeTrigger {
			if trigger, err := registry.GetTrigger(node.ComponentName()); err == nil {
				fields = trigger.Configuration()
			}
		}
		for _, name := range configuredIntegrationNames(fields, node.Configuration.Data()) {
			if id, ok := byName[name]; ok {
				usage.integrationIDs[id] = true
			}
		}
	}
	monitors, err := factory.RepositoryMonitoringWebhookIDs(tx)
	if err != nil {
		return nil, err
	}
	for _, id := range monitors {
		usage.webhookIDs[id.String()] = true
	}
	return usage, nil
}

// Follow typed fields, including runner credentials and imported environments.
// Arbitrary payloads and inactive credential choices do not imply usage.
func configuredIntegrationNames(fields []configuration.Field, values map[string]any) []string {
	var names []string
	for _, field := range fields {
		if !integrationFieldVisible(field, fields, values) {
			continue
		}
		value := values[field.Name]
		if field.Type == configuration.FieldTypeIntegration {
			if ref, err := configuration.DecodeIntegrationRef(value); err == nil && ref.IsSet() {
				names = append(names, ref.Name)
			}
			continue
		}
		if field.TypeOptions == nil {
			continue
		}
		if field.Type == configuration.FieldTypeObject && field.TypeOptions.Object != nil {
			object, _ := value.(map[string]any)
			names = append(names, configuredIntegrationNames(field.TypeOptions.Object.Schema, object)...)
		}
		if field.Type == configuration.FieldTypeList && field.TypeOptions.List != nil && field.TypeOptions.List.ItemDefinition != nil {
			items, _ := value.([]any)
			for _, item := range items {
				object, _ := item.(map[string]any)
				names = append(names, configuredIntegrationNames(field.TypeOptions.List.ItemDefinition.Schema, object)...)
			}
		}
	}
	return names
}

func integrationFieldVisible(field configuration.Field, siblings []configuration.Field, values map[string]any) bool {
	if len(field.VisibilityConditions) == 0 {
		return true
	}
	for _, condition := range field.VisibilityConditions {
		value := values[condition.Field]
		if value == nil {
			for _, sibling := range siblings {
				if sibling.Name == condition.Field {
					value = sibling.Default
					break
				}
			}
		}
		text := ""
		if value != nil {
			text = fmt.Sprint(value)
		}
		if !slices.Contains(condition.Values, text) && !(text != "" && slices.Contains(condition.Values, "*")) {
			return false
		}
	}
	return true
}
