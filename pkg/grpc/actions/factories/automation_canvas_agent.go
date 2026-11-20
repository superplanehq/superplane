package factories

import (
	"errors"
	"strings"

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

func automationAgentSettings(spec *automationspec.Agent) *pb.FactoryAppTemplateAgent {
	if spec == nil {
		return nil
	}
	agent := &pb.FactoryAppTemplateAgent{Component: automationHarnesses[spec.Harness], Model: spec.Model}
	if spec.Credentials != nil {
		agent.CredentialSource = string(spec.Credentials.Source)
		agent.CredentialIntegrationName = spec.Credentials.Name
	}
	return agent
}

func resolveAutomationAgent(tx *gorm.DB, factory *models.Factory, source *automationspec.Document) (*intakeAgent, error) {
	settings := automationAgentSettings(source.Agent)
	agent, err := customAutomationAgent(tx, factory, settings)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, source.Diagnostic("agent", "Configure an agent for this factory or select a ready agent connection.")
	}

	if settings.GetCredentialSource() == runner.CredentialsSourceIntegration {
		connection, err := models.FindIntegrationByName(tx, factory.OrganizationID, strings.TrimSpace(settings.GetCredentialIntegrationName()))
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if connection == nil || connection.State != models.IntegrationStateReady {
			return nil, source.Diagnostic("agent.credentials.name", "Select a ready connection in this organization.")
		}
		configured := intakeAgentFromIntegration(connection)
		if configured == nil || configured.Component != agent.Component {
			return nil, source.Diagnostic("agent.credentials.name", "Select a connection that supports this agent.")
		}
		agent.LLMProvider = configured.LLMProvider
	}

	if agent.subscription() {
		if !factory.OnboardingConfigValue().CodexSubscription {
			return nil, source.Diagnostic("agent.credentials.source", "Configure a ChatGPT subscription for this factory.")
		}
		if settings.GetModel() != "" {
			return nil, source.Diagnostic("agent.model", "Remove the model override when you use a ChatGPT subscription.")
		}
		return agent, nil
	}
	if agent.Component == models.SuperPlaneRunnerComponent {
		_, err := models.ResolveRunnableHostedLLMModel(tx, factory.OrganizationID, &factory.ID)
		if errors.Is(err, models.ErrSuperPlaneRunnerNoModel) || errors.Is(err, models.ErrSuperPlaneRunnerModelNotAllowed) {
			return nil, source.Diagnostic("agent", "Enable a SuperPlane-hosted model for this factory.")
		}
		return agent, err
	}
	config := map[string]any{"credentials": agent.credentials(), "model": agent.model()}
	agent.applyLLMProvider(config)
	if agent.Credentials["source"] == runner.CredentialsSourceIntegration || agent.Credentials["source"] == runner.CredentialsSourceSecret {
		if settings.GetModel() == "" {
			// Omitted YAML models must not inherit a fixed default outside the current policy.
			config["model"] = ""
		}
		model, err := models.ResolveBYOKRunnerModel(tx, factory.OrganizationID, &factory.ID, agent.Component, config)
		if errors.Is(err, models.ErrSelectableLLMModelNotAllowed) {
			return nil, source.Diagnostic("agent.model", "Select a model available for this agent in the factory.")
		}
		if err != nil {
			return nil, err
		}
		agent.Model = model
	}
	allowed, err := models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, agent.Component, config, agent.model())
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, source.Diagnostic("agent.model", "Select a model available for this agent in the factory.")
	}
	return agent, nil
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
