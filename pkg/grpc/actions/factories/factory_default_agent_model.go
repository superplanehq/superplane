package factories

import (
	"errors"
	"slices"
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

type agentConnectionReader interface {
	findIntegration(orgID, id uuid.UUID) (*models.Integration, error)
	findReady(orgID uuid.UUID, provider string) (*models.Integration, error)
}

type dbAgentConnections struct {
	tx *gorm.DB
}

func (c dbAgentConnections) findIntegration(orgID, id uuid.UUID) (*models.Integration, error) {
	integration, err := models.FindIntegrationInTransaction(c.tx, orgID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return integration, nil
}

func (c dbAgentConnections) findReady(orgID uuid.UUID, provider string) (*models.Integration, error) {
	return models.FindReadyBYOKIntegration(c.tx, orgID, provider)
}

type loadedAgentConnections struct {
	byID        map[uuid.UUID]models.Integration
	oldestReady map[agentConnectionKey]models.Integration
}

type agentConnectionKey struct {
	org uuid.UUID
	app string
}

func (c loadedAgentConnections) findIntegration(orgID, id uuid.UUID) (*models.Integration, error) {
	integration, ok := c.byID[id]
	if !ok || integration.OrganizationID != orgID {
		return nil, nil
	}
	return &integration, nil
}

func (c loadedAgentConnections) findReady(orgID uuid.UUID, provider string) (*models.Integration, error) {
	appName, err := models.BYOKIntegrationAppName(provider)
	if err != nil {
		return nil, err
	}
	integration, ok := c.oldestReady[agentConnectionKey{org: orgID, app: appName}]
	if !ok {
		return nil, nil
	}
	return &integration, nil
}

func loadAgentConnections(tx *gorm.DB, factories []models.Factory) (agentConnectionReader, error) {
	organizationIDs := make([]uuid.UUID, 0, 1)
	seenOrgs := make(map[uuid.UUID]struct{})
	integrationIDs := make([]uuid.UUID, 0)
	seenIDs := make(map[uuid.UUID]struct{})
	needsLookup := false
	for i := range factories {
		if !factoryLooksUpAgentConnection(&factories[i]) {
			continue
		}
		needsLookup = true
		orgID := factories[i].OrganizationID
		if _, seen := seenOrgs[orgID]; !seen {
			seenOrgs[orgID] = struct{}{}
			organizationIDs = append(organizationIDs, orgID)
		}
		id, ok := parsedAgentIntegrationID(factories[i].OnboardingConfigValue().AgentIntegrationID)
		if !ok {
			continue
		}
		if _, seen := seenIDs[id]; seen {
			continue
		}
		seenIDs[id] = struct{}{}
		integrationIDs = append(integrationIDs, id)
	}
	index := loadedAgentConnections{
		byID:        map[uuid.UUID]models.Integration{},
		oldestReady: map[agentConnectionKey]models.Integration{},
	}
	if !needsLookup {
		return index, nil
	}

	integrations, err := models.ListIntegrationsByIDOrReadyApp(tx, organizationIDs, integrationIDs, agentConnectionApps)
	if err != nil {
		return nil, err
	}
	for _, integration := range integrations {
		index.byID[integration.ID] = integration
		if integration.State != models.IntegrationStateReady || !slices.Contains(agentConnectionApps, integration.AppName) {
			continue
		}
		key := agentConnectionKey{org: integration.OrganizationID, app: integration.AppName}
		if _, exists := index.oldestReady[key]; exists {
			continue
		}
		index.oldestReady[key] = integration
	}
	return index, nil
}

func factoryLooksUpAgentConnection(factory *models.Factory) bool {
	switch strings.TrimSpace(factory.OnboardingConfigValue().AgentHarness) {
	case models.FactoryOnboardingAgentHarnessSuperPlane,
		models.FactoryOnboardingAgentHarnessCodex,
		models.FactoryOnboardingAgentHarnessCursor:
		return false
	default:
		return true
	}
}

func parsedAgentIntegrationID(raw string) (uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

var agentConnectionApps = []string{"claude", "openai", "openrouter", models.CustomLLMAppName}

func activeFactoryModelSource(tx *gorm.DB, factory *models.Factory) (string, error) {
	return resolveFactoryModelSource(dbAgentConnections{tx: tx}, factory)
}

func resolveFactoryModelSource(connections agentConnectionReader, factory *models.Factory) (string, error) {
	config := factory.OnboardingConfigValue()
	switch strings.TrimSpace(config.AgentHarness) {
	case models.FactoryOnboardingAgentHarnessSuperPlane:
		return modelSourceHosted, nil
	case models.FactoryOnboardingAgentHarnessCodex:
		return modelSourceOpenAI, nil
	case models.FactoryOnboardingAgentHarnessCursor:
		return "", nil
	case models.FactoryOnboardingAgentHarnessClaudeCode:
		return claudeFactoryModelSource(connections, factory, config.AgentIntegrationID)
	default:
		return unspecifiedFactoryModelSource(connections, factory, config.AgentIntegrationID)
	}
}

func claudeFactoryModelSource(connections agentConnectionReader, factory *models.Factory, integrationID string) (string, error) {
	source, err := modelSourceFromAgentIntegration(connections, factory, integrationID)
	if err != nil || source != "" {
		return source, err
	}
	return firstReadyModelSource(connections, factory.OrganizationID, []string{
		modelSourceAnthropic,
		modelSourceOpenRouter,
		modelSourceCustom,
	})
}

func unspecifiedFactoryModelSource(connections agentConnectionReader, factory *models.Factory, integrationID string) (string, error) {
	source, err := modelSourceFromAgentIntegration(connections, factory, integrationID)
	if err != nil || source != "" {
		return source, err
	}
	source, err = firstReadyModelSource(connections, factory.OrganizationID, []string{
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

func modelSourceFromAgentIntegration(connections agentConnectionReader, factory *models.Factory, integrationID string) (string, error) {
	id, ok := parsedAgentIntegrationID(integrationID)
	if !ok {
		return "", nil
	}
	integration, err := connections.findIntegration(factory.OrganizationID, id)
	if err != nil || integration == nil {
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

func firstReadyModelSource(connections agentConnectionReader, orgID uuid.UUID, sources []string) (string, error) {
	for _, source := range sources {
		integration, err := connections.findReady(orgID, source)
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
