package factories

import (
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// intakeAgentSpec pairs a BYOK runner with its integration. List order is the
// intake preference.
type intakeAgentSpec struct {
	component      string
	integrationApp string
	model          string
	llmProvider    string
}

var intakeAgentSpecs = []intakeAgentSpec{
	{
		component:      "runnerClaudeCode",
		integrationApp: "claude",
		model:          "claude-opus-5-5",
	},
	{
		component:      "runnerCodex",
		integrationApp: "openai",
		model:          "gpt-5",
	},
	{
		component:      "runnerOpenRouter",
		integrationApp: "openrouter",
		model:          "anthropic/claude-sonnet-4-6",
	},
	{
		component:      "runnerOpenRouter",
		integrationApp: models.CustomLLMAppName,
		llmProvider:    models.UsageProviderCustom,
	},
}

// intakeAgent is the runner a generated analysis node uses.
type intakeAgent struct {
	Component   string
	Credentials map[string]any
	Model       string
	LLMProvider string
}

func (a *intakeAgent) component() string {
	if a == nil || a.Component == "" {
		return intakeAgentSpecs[0].component
	}
	return a.Component
}

func (a *intakeAgent) credentials() map[string]any {
	if a == nil {
		return nil
	}
	return a.Credentials
}

func (a *intakeAgent) model() string {
	if a != nil && a.Component == models.SuperPlaneRunnerComponent {
		return ""
	}
	if a != nil && a.Model != "" {
		return a.Model
	}
	if a != nil && a.LLMProvider == models.UsageProviderCustom {
		return ""
	}
	return defaultIntakeAgentModel(a.component())
}

func defaultIntakeAgentModel(component string) string {
	index := slices.IndexFunc(intakeAgentSpecs, func(spec intakeAgentSpec) bool {
		return spec.component == component
	})
	if index < 0 {
		return ""
	}
	return intakeAgentSpecs[index].model
}

// resolveIntakeAgent picks the workspace agent. A SuperPlane harness from
// setup uses Run SuperPlane Agent and ignores organization BYOK keys.
// Older workspaces with no harness still fall back to those installations.
func resolveIntakeAgent(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	config := factory.OnboardingConfigValue()
	if agent := fillCustomProviderModel(tx, factory, intakeAgentFromSetup(tx, factory, config.AgentIntegrationID)); agent != nil {
		return agent
	}

	if config.AgentHarness == models.FactoryOnboardingAgentHarnessSuperPlane {
		return intakeAgentFromHostedProvider(tx, factory)
	}

	if agent := fillCustomProviderModel(tx, factory, intakeAgentFromInstallations(tx, factory)); agent != nil {
		return agent
	}

	return intakeAgentFromHostedProvider(tx, factory)
}

// intakeAgentFromSetup reads the agent recorded in workspace setup.
func intakeAgentFromSetup(tx *gorm.DB, factory *models.Factory, integrationID string) *intakeAgent {
	if strings.TrimSpace(integrationID) == "" {
		return nil
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		log.Warnf("factory %s: intake ignores the setup agent, invalid integration id %q", factory.ID, integrationID)
		return nil
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		log.Warnf("factory %s: intake ignores the setup agent, integration %s not found: %v", factory.ID, id, err)
		return nil
	}

	return intakeAgentFromIntegration(integration)
}

// intakeAgentFromInstallations takes the first ready agent installation.
func intakeAgentFromInstallations(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	integrations, err := models.ListIntegrations(tx, factory.OrganizationID)
	if err != nil {
		log.Warnf("factory %s: intake cannot read the installations of the organization: %v", factory.ID, err)
		return nil
	}

	for _, spec := range intakeAgentSpecs {
		for i := range integrations {
			if integrations[i].AppName != spec.integrationApp {
				continue
			}
			if agent := intakeAgentFromIntegration(&integrations[i]); agent != nil {
				return agent
			}
		}
	}

	return nil
}

// intakeAgentFromHostedProvider uses Run SuperPlane Agent when no integration
// is available and the instance has a SuperPlane agent model.
func intakeAgentFromHostedProvider(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	defaultModel, err := models.GetInstallationDefaultHostedLLMModel(tx)
	if err != nil {
		log.Warnf("factory %s: intake cannot read the SuperPlane agent model: %v", factory.ID, err)
		return nil
	}
	if !defaultModel.IsSet() {
		return nil
	}
	if err := models.AssertDefaultHostedLLMModelAllowed(tx, defaultModel); err != nil {
		return nil
	}

	return &intakeAgent{
		Component: models.SuperPlaneRunnerComponent,
	}
}

func resolveGitHubInstallationName(tx *gorm.DB, factory *models.Factory) string {
	if tx == nil || factory == nil {
		return intakeGitHubAppName
	}

	if strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID) != "" {
		if name := githubInstallationNameFromVCS(tx, factory); name != "" {
			return name
		}
		return intakeGitHubAppName
	}

	integrations, err := models.ListIntegrations(tx, factory.OrganizationID)
	if err != nil {
		log.Warnf("factory %s: backlog cannot read GitHub installations: %v", factory.ID, err)
		return intakeGitHubAppName
	}

	for i := range integrations {
		if integrations[i].AppName != intakeGitHubAppName {
			continue
		}
		if integrations[i].State != models.IntegrationStateReady {
			continue
		}
		if name := strings.TrimSpace(integrations[i].InstallationName); name != "" {
			return name
		}
	}

	return intakeGitHubAppName
}

func githubInstallationNameFromVCS(tx *gorm.DB, factory *models.Factory) string {
	integrationID := strings.TrimSpace(factory.OnboardingConfigValue().VCSIntegrationID)
	if integrationID == "" {
		return ""
	}

	integration := findIntakeGitHubIntegration(tx, factory, integrationID)
	if integration == nil || integration.State != models.IntegrationStateReady {
		return ""
	}

	return strings.TrimSpace(integration.InstallationName)
}

func intakeAgentFromIntegration(integration *models.Integration) *intakeAgent {
	if integration.State != models.IntegrationStateReady {
		return nil
	}

	index := slices.IndexFunc(intakeAgentSpecs, func(spec intakeAgentSpec) bool {
		return spec.integrationApp == integration.AppName
	})
	if index < 0 {
		return nil
	}

	return &intakeAgent{
		Component: intakeAgentSpecs[index].component,
		Credentials: map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": integration.InstallationName},
		},
		Model:       intakeAgentSpecs[index].model,
		LLMProvider: intakeAgentSpecs[index].llmProvider,
	}
}

func (a *intakeAgent) applyLLMProvider(configuration map[string]any) {
	if a == nil || configuration == nil || a.LLMProvider != models.UsageProviderCustom {
		return
	}
	configuration["llmProvider"] = models.UsageProviderCustom
}

func fillCustomProviderModel(tx *gorm.DB, factory *models.Factory, agent *intakeAgent) *intakeAgent {
	if agent == nil || agent.LLMProvider != models.UsageProviderCustom || strings.TrimSpace(agent.Model) != "" {
		return agent
	}
	if model := customModelFromFactory(tx, factory); model != "" {
		agent.Model = model
		return agent
	}
	if tx == nil || factory == nil {
		return agent
	}
	ids, err := models.ResolveSelectableLLMModels(tx, factory.OrganizationID, &factory.ID, models.UsageProviderCustom, models.UsageFundingSourceBYOK)
	if err != nil || len(ids) == 0 {
		return agent
	}
	sort.Strings(ids)
	agent.Model = ids[0]
	return agent
}

func customModelFromFactory(tx *gorm.DB, factory *models.Factory) string {
	if tx == nil || factory == nil {
		return ""
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return ""
	}
	for i := range canvases {
		if canvases[i].LiveVersionID == nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionInTransaction(tx, canvases[i].ID)
		if err != nil {
			continue
		}
		for _, node := range version.Nodes {
			if node.ComponentName() != "runnerOpenRouter" {
				continue
			}
			provider, _ := node.Configuration["llmProvider"].(string)
			if provider != models.UsageProviderCustom {
				continue
			}
			if model := strings.TrimSpace(configString(node.Configuration, "model")); model != "" {
				return model
			}
		}
	}
	return ""
}
