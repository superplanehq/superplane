package models

import (
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RunnerAcceptsModel checks the provider and funding source of the executing node.
func RunnerAcceptsModel(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, component string, configuration map[string]any, model string, ownerIDs ...uuid.UUID) (bool, error) {
	if component == SuperPlaneRunnerComponent {
		parsed, err := ParseHostedLLMModelKey(model)
		if err != nil || !parsed.IsSet() {
			return false, nil
		}
		return ModelIsSelectable(tx, orgID, factoryID, parsed.Provider, UsageFundingSourceHosted, parsed.Model)
	}
	provider := ""
	switch component {
	case "runnerClaudeCode":
		provider = UsageProviderAnthropic
	case "runnerCodex":
		provider = UsageProviderOpenAI
	case "runnerOpenRouter":
		provider = UsageProviderOpenRouter
		if configuration["llmProvider"] == UsageProviderCustom {
			provider = UsageProviderCustom
		}
	default:
		return false, nil
	}
	credentials, _ := configuration["credentials"].(map[string]any)
	source, _ := credentials["source"].(string)
	if strings.TrimSpace(source) == "subscription" {
		if component != "runnerCodex" || factoryID == nil || *factoryID == uuid.Nil {
			return false, nil
		}
		if len(ownerIDs) != 1 {
			return false, nil
		}
		models, err := FactoryCodexSubscriptionModels(tx, orgID, *factoryID, ownerIDs[0])
		return slices.Contains(models, strings.TrimSpace(model)), err
	}
	funding := UsageFundingSourceHosted
	switch strings.TrimSpace(source) {
	case "secret", "integration":
		funding = UsageFundingSourceBYOK
	}
	return ModelIsSelectable(tx, orgID, factoryID, provider, funding, model)
}
