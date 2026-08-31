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

// intakeAgentSpec pairs one runner component with the installation and the
// SuperPlane-hosted provider that can authenticate it. The order of the list is
// the order an intake prefers, and it follows the order the workspace setup
// offers the providers in.
type intakeAgentSpec struct {
	component      string
	integrationApp string
	hostedProvider string
	// hostedModelHint is the part of a model id an intake looks for first on a
	// hosted allowlist. An allowlist without it falls back to its first model.
	hostedModelHint string
	// model is the model the generated nodes name. A runner has a default of
	// its own, but a node that leaves the field empty shows the user no model
	// and runs on whatever the agent CLI picks, so each spec names one.
	//
	// The Backlog scorer and the PR feedback runner weigh evidence rather than
	// write code: the analysis scores a work order, and the PR feedback runner
	// judges which review requests are safe to apply. Anthropic therefore
	// reaches for Opus, the way the planning agent of the factory line does,
	// while the implementation agent stays on Sonnet.
	model string
}

var intakeAgentSpecs = []intakeAgentSpec{
	{
		component:       "runnerClaudeCode",
		integrationApp:  "claude",
		hostedProvider:  models.UsageProviderAnthropic,
		hostedModelHint: "opus",
		model:           "opus",
	},
	{
		component:       "runnerCodex",
		integrationApp:  "openai",
		hostedProvider:  models.UsageProviderOpenAI,
		hostedModelHint: "gpt-5",
		model:           "gpt-5",
	},
	{
		component:       "runnerOpenRouter",
		integrationApp:  "openrouter",
		hostedProvider:  models.UsageProviderOpenRouter,
		hostedModelHint: "sonnet",
		model:           "anthropic/claude-sonnet-4-6",
	},
}

// intakeAgent is the runner the generated analysis node scores with, together
// with the credentials it authenticates by. A runner node without credentials
// cannot run, so an intake takes the agent the workspace already uses.
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

// model is the model the generated node names. An agent that carries no model
// of its own takes the default of its runner, so the node always shows the
// model the analysis runs on.
func (a *intakeAgent) model() string {
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

// resolveIntakeAgent picks the runner and the credentials of the Backlog
// analysis node. The workspace setup records the coding agent of the
// workspace, so scoring uses the same one. A workspace that named no agent
// still has its installations and the hosted providers to fall back on. A
// graph that finds nothing keeps an incomplete analysis node, and the user
// completes it in the canvas.
func resolveIntakeAgent(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	config := factory.OnboardingConfigValue()
	if agent := intakeAgentFromPersistedWorkspaceAgent(tx, factory, config); agent != nil {
		return agent
	}
	if agent := intakeAgentFromSetup(tx, factory, config.AgentIntegrationID); agent != nil {
		return agent
	}

	if agent := intakeAgentFromInstallations(tx, factory); agent != nil {
		return agent
	}

	return intakeAgentFromHostedProvider(tx, factory)
}

// intakeAgentFromPersistedWorkspaceAgent keeps newly created generated
// automations on the agent selected in workspace settings. Older factories do
// not have AgentProvider, so they retain the legacy discovery fallback below.
func intakeAgentFromPersistedWorkspaceAgent(
	tx *gorm.DB,
	factory *models.Factory,
	config models.FactoryOnboardingConfig,
) *intakeAgent {
	spec, ok := factoryAgentProviderSpecs[config.AgentProvider]
	if !ok {
		return nil
	}
	model := config.AgentModel
	if model == "" {
		model = spec.model
	}
	if config.AgentIntegrationID == "" {
		return &intakeAgent{
			Component:   spec.component,
			Credentials: map[string]any{"source": runner.CredentialsSourceHosted},
			Model:       model,
		}
	}

	integrationID, err := uuid.Parse(config.AgentIntegrationID)
	if err != nil {
		return nil
	}
	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, integrationID)
	if err != nil || integration.State != models.IntegrationStateReady || integration.AppName != spec.integrationApp {
		return nil
	}
	return &intakeAgent{
		Component: spec.component,
		Credentials: map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": integration.InstallationName},
		},
		Model: model,
	}
}

// intakeAgentFromSetup reads the agent the workspace setup recorded. A setup
// that used hosted credentials records no installation, and a setup that used
// an agent without a runner component (such as Cursor) records one an intake
// cannot score with, so both fall through to the other sources.
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

// intakeAgentFromInstallations takes the first agent installation the
// organization has, in the order an intake prefers. A workspace set up before
// setup recorded its agent reaches its agent this way.
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

// intakeAgentFromHostedProvider falls back to SuperPlane-hosted credentials.
// A hosted run needs a model from the allowlist of the provider, so a provider
// without one cannot serve the intake.
func intakeAgentFromHostedProvider(tx *gorm.DB, factory *models.Factory) *intakeAgent {
	providers, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		log.Warnf("factory %s: intake cannot read the hosted providers: %v", factory.ID, err)
		return nil
	}

	for _, spec := range intakeAgentSpecs {
		index := slices.IndexFunc(providers, func(provider models.HostedLLMProvider) bool {
			return provider.Provider == spec.hostedProvider && provider.OffersHostedModels()
		})
		if index < 0 {
			continue
		}

		model := hostedIntakeModel(providers[index], spec.hostedModelHint)
		if model == "" {
			continue
		}

		return &intakeAgent{
			Component:   spec.component,
			Credentials: map[string]any{"source": runner.CredentialsSourceHosted},
			Model:       model,
		}
	}

	return nil
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

// hostedIntakeModel picks the model an intake scores with from an allowlist.
// The list is free-form, so the hint only expresses a preference: any
// allowlisted model can run the analysis. The choice is stable, because an
// intake that is created twice has to produce the same graph.
func hostedIntakeModel(provider models.HostedLLMProvider, hint string) string {
	allowed := []string{}
	for _, model := range provider.AllowedModels {
		if trimmed := strings.TrimSpace(model); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	if len(allowed) == 0 {
		return ""
	}

	slices.Sort(allowed)
	for _, model := range allowed {
		if strings.Contains(strings.ToLower(model), hint) {
			return model
		}
	}

	return allowed[0]
}
