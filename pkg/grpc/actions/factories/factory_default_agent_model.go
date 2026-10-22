package factories

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func setFactoryDefaultAgentModel(tx *gorm.DB, factory *models.Factory, model, provider string) error {
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	if model == "" {
		return invalidArgument("default model is required")
	}
	if err := rejectUnusableFactoryDefault(tx, factory, model, provider); err != nil {
		return err
	}
	return factory.UpdateDefaultAgentModel(tx, model, provider)
}

func rejectUnusableFactoryDefault(tx *gorm.DB, factory *models.Factory, model, provider string) error {
	source, err := activeFactoryModelSource(tx, factory)
	if err != nil {
		return err
	}
	listed, err := models.ListSelectableLLMModels(tx, factory.OrganizationID, &factory.ID)
	if err != nil {
		return err
	}
	if !selectableDefaultForSource(listed, model, provider, source) {
		return models.ErrSelectableLLMModelNotAllowed
	}
	return nil
}

func clearStaleFactoryDefaultModel(tx *gorm.DB, factory *models.Factory, source string) error {
	if strings.TrimSpace(factory.DefaultAgentModel) == "" {
		return nil
	}
	allowed, err := factoryDefaultSelectableForSource(tx, factory, source)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	return factory.UpdateDefaultAgentModel(tx, "", "")
}

func factoryDefaultSelectableForSource(tx *gorm.DB, factory *models.Factory, source string) (bool, error) {
	listed, err := models.ListSelectableLLMModels(tx, factory.OrganizationID, &factory.ID)
	if err != nil {
		return false, err
	}
	return selectableDefaultForSource(listed, factory.DefaultAgentModel, factory.DefaultAgentModelProvider, source), nil
}

func activeFactoryModelSource(tx *gorm.DB, factory *models.Factory) (string, error) {
	config := factory.OnboardingConfigValue()
	switch strings.TrimSpace(config.AgentHarness) {
	case models.FactoryOnboardingAgentHarnessSuperPlane:
		return modelSourceHosted, nil
	case models.FactoryOnboardingAgentHarnessCodex:
		return modelSourceOpenAI, nil
	case models.FactoryOnboardingAgentHarnessCursor:
		return "", nil
	case models.FactoryOnboardingAgentHarnessClaudeCode:
		return claudeFactoryModelSource(tx, factory, config.AgentIntegrationID)
	default:
		return unspecifiedFactoryModelSource(tx, factory, config.AgentIntegrationID)
	}
}

func claudeFactoryModelSource(tx *gorm.DB, factory *models.Factory, integrationID string) (string, error) {
	source, err := modelSourceFromAgentIntegration(tx, factory, integrationID)
	if err != nil || source != "" {
		return source, err
	}
	return firstReadyModelSource(tx, factory.OrganizationID, []string{
		modelSourceAnthropic,
		modelSourceOpenRouter,
		modelSourceCustom,
	})
}

func unspecifiedFactoryModelSource(tx *gorm.DB, factory *models.Factory, integrationID string) (string, error) {
	source, err := modelSourceFromAgentIntegration(tx, factory, integrationID)
	if err != nil || source != "" {
		return source, err
	}
	source, err = firstReadyModelSource(tx, factory.OrganizationID, []string{
		modelSourceAnthropic,
		modelSourceOpenAI,
		modelSourceOpenRouter,
		modelSourceCustom,
	})
	if err != nil || source != "" {
		return source, err
	}
	return modelSourceHosted, nil
}

func modelSourceFromAgentIntegration(tx *gorm.DB, factory *models.Factory, integrationID string) (string, error) {
	integrationID = strings.TrimSpace(integrationID)
	if integrationID == "" {
		return "", nil
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return "", nil
	}
	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return modelSourceFromAppName(integration.AppName), nil
}

func modelSourceFromAppName(appName string) string {
	switch appName {
	case "claude":
		return modelSourceAnthropic
	case "openai":
		return modelSourceOpenAI
	case "openrouter":
		return modelSourceOpenRouter
	case models.CustomLLMAppName:
		return modelSourceCustom
	default:
		return ""
	}
}

func firstReadyModelSource(tx *gorm.DB, orgID uuid.UUID, sources []string) (string, error) {
	for _, source := range sources {
		integration, err := models.FindReadyBYOKIntegration(tx, orgID, source)
		if err != nil {
			return "", err
		}
		if integration != nil {
			return source, nil
		}
	}
	return "", nil
}

func selectableDefaultForSource(listed []models.SelectableLLMModel, model, provider, source string) bool {
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	source = strings.TrimSpace(source)
	if model == "" {
		return false
	}
	for _, item := range listed {
		if item.Model.ID != model {
			continue
		}
		if source == modelSourceHosted {
			if provider != "" && item.Source.ID == models.UsageFundingSourceHosted && item.Provider.ID == provider {
				return true
			}
			continue
		}
		if provider == "" && item.Source.ID == models.UsageFundingSourceBYOK && item.Provider.ID == source {
			return true
		}
	}
	return false
}
