package factories

import (
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

var automationHarnesses = map[automationspec.Harness]string{
	automationspec.SuperPlane: models.SuperPlaneRunnerComponent,
	automationspec.ClaudeCode: "runnerClaudeCode",
	automationspec.Codex:      "runnerCodex",
	automationspec.OpenRouter: "runnerOpenRouter",
}

// validateCustomAutomationModel uses the same catalog as the model picker.
// A subscription's catalog belongs to the member editing the automation.
func validateCustomAutomationModel(tx *gorm.DB, factory *models.Factory, agent *intakeAgent, callerID uuid.UUID) (bool, error) {
	config := map[string]any{"credentials": agent.credentials()}
	agent.applyLLMProvider(config)
	if agent.subscription() {
		member, err := models.FindActiveUserByIDInTransaction(tx, factory.OrganizationID.String(), callerID.String())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if member.Type != models.UserTypeHuman {
			return false, nil
		}
		return models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, agent.Component, config, agent.Model, callerID)
	}
	return models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, agent.Component, config, agent.Model)
}

func automationAgentSettings(tx *gorm.DB, factory *models.Factory, document *automationspec.Document) (*pb.FactoryAppTemplateAgent, error) {
	spec := document.Agent
	if spec == nil {
		return nil, nil
	}
	agent := &pb.FactoryAppTemplateAgent{Component: automationHarnesses[spec.Harness], Model: spec.Model}
	if spec.Harness == automationspec.SuperPlane && spec.Model != "" {
		model, err := resolveAutomationHostedModel(tx, factory, document, "")
		if err != nil {
			return nil, err
		}
		agent.Model = model
	}
	if spec.Credentials != nil {
		agent.CredentialSource = string(spec.Credentials.Source)
		agent.CredentialIntegrationName = spec.Credentials.Name
	}
	return agent, nil
}

func configuredAutomationAgent(tx *gorm.DB, factory *models.Factory, live *models.CanvasVersion) (*intakeAgent, string) {
	workspace := resolveIntakeAgent(tx, factory)
	if live == nil {
		return workspace, ""
	}
	current := intakeAgentFromCanvasNodes(live.Nodes)
	if current == nil {
		return nil, ""
	}
	savedModel := current.Model
	if workspace != nil && workspace.Component == current.Component && reflect.DeepEqual(workspace.Credentials, current.Credentials) {
		return workspace, savedModel
	}
	// Keep the saved source. An omitted model uses that source's default.
	current.Model = ""
	return current, savedModel
}

func resolveAutomationAgent(tx *gorm.DB, factory *models.Factory, source *automationspec.Document, live *models.CanvasVersion, callerID uuid.UUID) (*intakeAgent, error) {
	agent, savedModel := configuredAutomationAgent(tx, factory, live)
	if agent == nil {
		if live != nil {
			return nil, unrepresentableAutomationCanvas()
		}
		return nil, source.Diagnostic("agent", "Configure an agent for this workspace.")
	}
	configured, err := automationAgentDefinition(agent)
	if err != nil {
		return nil, err
	}
	settings := source.Agent
	if settings != nil {
		if settings.Harness != "" && settings.Harness != configured.Harness {
			return nil, source.Diagnostic("agent.harness", "Use the configured agent. Change the model source in workspace settings.")
		}
		if settings.Credentials != nil && (configured.Credentials == nil || *settings.Credentials != *configured.Credentials) {
			return nil, source.Diagnostic("agent.credentials", "Use the configured credentials. Change the model source in workspace settings.")
		}
	}
	if credentials := configured.Credentials; credentials != nil && credentials.Source == automationspec.Integration {
		connection, err := models.FindIntegrationByName(tx, factory.OrganizationID, credentials.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if connection == nil || connection.State != models.IntegrationStateReady {
			return nil, source.Diagnostic("agent", "Reconnect the configured agent in this workspace.")
		}
		installed := intakeAgentFromIntegration(connection)
		if installed == nil || installed.Component != agent.Component {
			return nil, source.Diagnostic("agent", "The configured connection must support this agent.")
		}
		agent.LLMProvider = installed.LLMProvider
	}
	if agent.subscription() && !factory.OnboardingConfigValue().CodexSubscription {
		return nil, source.Diagnostic("agent", "Configure a ChatGPT subscription for this workspace.")
	}
	model := ""
	if settings != nil {
		model = strings.TrimSpace(settings.Model)
	}
	if model == "" && agent.subscription() {
		return agent, nil // The task owner's subscription supplies the default at run time.
	}
	if model == "" && agent.Component == models.SuperPlaneRunnerComponent {
		_, err := models.ResolveRunnableHostedLLMModel(tx, factory.OrganizationID, &factory.ID)
		if errors.Is(err, models.ErrSuperPlaneRunnerNoModel) || errors.Is(err, models.ErrSuperPlaneRunnerModelNotAllowed) {
			return nil, source.Diagnostic("agent", "Enable a SuperPlane-hosted model for this workspace.")
		}
		return agent, err
	}
	if agent.Component == models.SuperPlaneRunnerComponent {
		model, err = resolveAutomationHostedModel(tx, factory, source, savedModel)
		if err != nil {
			return nil, err
		}
	}
	if funding := agent.Credentials["source"]; funding == runner.CredentialsSourceIntegration || funding == runner.CredentialsSourceSecret {
		// Let the execution policy select a permitted BYOK default when YAML omits a model.
		config := map[string]any{"credentials": agent.credentials(), "model": model}
		agent.applyLLMProvider(config)
		resolved, err := models.ResolveBYOKRunnerModel(tx, factory.OrganizationID, &factory.ID, agent.Component, config)
		if errors.Is(err, models.ErrSelectableLLMModelNotAllowed) {
			return nil, source.Diagnostic("agent.model", "Select an available model for the configured agent.")
		}
		if err != nil {
			return nil, err
		}
		agent.Model = resolved
		return agent, nil
	}
	if model == "" {
		model = agent.model()
	}
	agent.Model = model
	allowed, err := validateCustomAutomationModel(tx, factory, agent, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, source.Diagnostic("agent.model", "Select an available model for the configured agent.")
	}
	return agent, nil
}

// Accept the provider-qualified YAML spelling used by older editors.
func automationCanvasHostedModel(model string) string {
	provider, id, _ := strings.Cut(model, "::")
	hosted, err := models.NormalizeDefaultHostedLLMModel(provider, id)
	if err != nil || !hosted.IsSet() {
		return model
	}
	return models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, hosted.Provider, hosted.Model)
}

func resolveAutomationHostedModel(tx *gorm.DB, factory *models.Factory, source *automationspec.Document, savedModel string) (string, error) {
	id := strings.TrimSpace(source.Agent.Model)
	if isQualifiedAutomationHostedModel(id) {
		return automationCanvasHostedModel(id), nil
	}
	// Keep the route pinned on the live canvas, even if a second provider now offers this ID.
	if saved, err := models.ParseHostedLLMModelKey(savedModel); err == nil && saved.Model == id {
		return models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, saved.Provider, id), nil
	}
	providers, err := automationHostedModelProviders(tx, factory.OrganizationID, factory.ID, id)
	if err != nil {
		return "", err
	}
	switch len(providers) {
	case 0:
		return "", source.Diagnostic("agent.model", "Select an available model for the configured agent.")
	case 1:
		return models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, providers[0], id), nil
	default:
		return "", source.Diagnostic("agent.model", "Multiple hosted providers offer this model ("+strings.Join(providers, ", ")+"). Use provider::model to choose one.")
	}
}

func isQualifiedAutomationHostedModel(model string) bool {
	prefix, _, found := strings.Cut(model, "::")
	if !found {
		return false
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	switch prefix {
	case models.UsageFundingSourceHosted, models.UsageFundingSourceBYOK, models.UsageFundingSourceSubscription:
		return true
	}
	return slices.Contains(models.KnownHostedLLMProviders(), prefix)
}

func automationHostedModelProviders(tx *gorm.DB, orgID, factoryID uuid.UUID, modelID string) ([]string, error) {
	var providers []string
	for _, provider := range models.KnownHostedLLMProviders() {
		allowed, err := models.ModelIsSelectable(tx, orgID, &factoryID, provider, models.UsageFundingSourceHosted, modelID)
		if err != nil {
			return nil, err
		}
		if allowed {
			providers = append(providers, provider)
		}
	}
	return providers, nil
}

func automationAgentDefinition(agent *intakeAgent) (*automationspec.Agent, error) {
	if agent == nil {
		return nil, unrepresentableAutomationCanvas()
	}
	for harness, component := range automationHarnesses {
		if component != agent.Component {
			continue
		}
		result := &automationspec.Agent{Harness: harness, Model: agent.Model}
		source := configString(agent.Credentials, "source")
		switch source {
		case "":
		case runner.CredentialsSourceHosted, runner.CredentialsSourceSubscription:
			result.Credentials = &automationspec.Credentials{Source: automationspec.CredentialSource(source)}
		case runner.CredentialsSourceIntegration:
			connection, _ := agent.Credentials["integration"].(map[string]any)
			result.Credentials = &automationspec.Credentials{Source: automationspec.Integration, Name: configString(connection, "name")}
		default:
			return nil, unrepresentableAutomationCanvas()
		}
		return result, nil
	}
	return nil, unrepresentableAutomationCanvas()
}

func exportAutomationAgent(tx *gorm.DB, orgID, factoryID uuid.UUID, agent *intakeAgent) (*automationspec.Agent, error) {
	spec, err := automationAgentDefinition(agent)
	if err != nil || spec.Harness != automationspec.SuperPlane || spec.Model == "" {
		return spec, err
	}
	hosted, err := models.ParseHostedLLMModelKey(spec.Model)
	if err != nil || !hosted.IsSet() {
		return spec, nil
	}
	spec.Model = models.FormatHostedLLMModelKey(hosted.Provider, hosted.Model)
	if isQualifiedAutomationHostedModel(hosted.Model) {
		return spec, nil
	}
	providers, err := automationHostedModelProviders(tx, orgID, factoryID, hosted.Model)
	if err != nil {
		return nil, err
	}
	if len(providers) == 1 && providers[0] == hosted.Provider {
		spec.Model = hosted.Model
	}
	return spec, nil
}
