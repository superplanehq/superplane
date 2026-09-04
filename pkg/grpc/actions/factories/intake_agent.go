package factories

import (
	"slices"
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
}

var intakeAgentSpecs = []intakeAgentSpec{
	{
		component:      "runnerClaudeCode",
		integrationApp: "claude",
		model:          "opus",
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
}

// intakeAgent is the runner a generated analysis node uses.
type intakeAgent struct {
	Component   string
	Credentials map[string]any
	Model       string
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

// resolveIntakeAgent picks the workspace agent. Credit without an installation
// uses Run SuperPlane Agent.
func resolveIntakeAgent(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	config := factory.OnboardingConfigValue()
	if agent := intakeAgentFromSetup(tx, factory, config.AgentIntegrationID); agent != nil {
		return agent
	}

	if agent := intakeAgentFromInstallations(tx, factory); agent != nil {
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
		Model: intakeAgentSpecs[index].model,
	}
}
