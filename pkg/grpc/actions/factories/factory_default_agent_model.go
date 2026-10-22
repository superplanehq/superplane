package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func setFactoryDefaultAgentModel(tx *gorm.DB, factory *models.Factory, model, provider string) error {
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	if model == "" {
		return invalidArgument("default model is required")
	}
	if _, ok, err := selectableFactoryDefault(tx, factory, model, provider); err != nil {
		return err
	} else if !ok {
		return models.ErrSelectableLLMModelNotAllowed
	}
	return factory.UpdateDefaultAgentModel(tx, model, provider)
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

func selectableFactoryDefault(tx *gorm.DB, factory *models.Factory, model, provider string) (models.SelectableLLMModel, bool, error) {
	listed, err := models.ListSelectableLLMModels(tx, factory.OrganizationID, &factory.ID)
	if err != nil {
		return models.SelectableLLMModel{}, false, err
	}
	found, ok := matchSelectableFactoryDefault(listed, model, provider)
	return found, ok, nil
}

func matchSelectableFactoryDefault(listed []models.SelectableLLMModel, model, provider string) (models.SelectableLLMModel, bool) {
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	if model == "" {
		return models.SelectableLLMModel{}, false
	}
	for _, item := range listed {
		if item.Model.ID != model {
			continue
		}
		if provider == "" && item.Source.ID == models.UsageFundingSourceBYOK {
			return item, true
		}
		if provider != "" && item.Source.ID == models.UsageFundingSourceHosted && item.Provider.ID == provider {
			return item, true
		}
	}
	return models.SelectableLLMModel{}, false
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
