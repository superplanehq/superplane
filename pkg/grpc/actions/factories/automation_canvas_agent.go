package factories

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var automationHarnesses = map[automationspec.Harness]string{
	automationspec.SuperPlane: models.SuperPlaneRunnerComponent,
	automationspec.ClaudeCode: "runnerClaudeCode",
	automationspec.Codex:      "runnerCodex",
	automationspec.OpenRouter: "runnerOpenRouter",
}

func resolveAutomationAgent(tx *gorm.DB, factory *models.Factory, source *automationspec.Document) (*intakeAgent, error) {
	var agent *intakeAgent
	var override string
	if source.Agent == nil {
		agent = resolveIntakeAgent(tx, factory)
	} else {
		component := automationHarnesses[source.Agent.Harness]
		override = strings.TrimSpace(source.Agent.Model)
		switch credentials := source.Agent.Credentials; {
		case component == models.SuperPlaneRunnerComponent:
			agent = &intakeAgent{Component: component}
		case credentials == nil:
			agent = resolveIntakeAgent(tx, factory)
			if agent == nil || agent.Component != component {
				return nil, source.Diagnostic("agent.credentials", "Select a connection for this agent or use the factory agent.")
			}
		case credentials.Source == automationspec.Subscription:
			agent = &intakeAgent{Component: component, Credentials: map[string]any{"source": runner.CredentialsSourceSubscription}}
		case credentials.Source == automationspec.Integration:
			var err error
			agent, err = findAutomationAgentConnection(tx, factory, component, credentials.Name)
			if err != nil {
				return nil, err
			}
			if agent == nil {
				return nil, source.Diagnostic("agent.credentials.name", "Select one ready connection in this organization that supports this agent.")
			}
		}
	}
	if agent == nil {
		return nil, source.Diagnostic("agent", "Configure an agent for this factory or select a ready agent connection.")
	}
	if agent.subscription() {
		if override != "" {
			return nil, source.Diagnostic("agent.model", "Remove the model override when you use a ChatGPT subscription.")
		}
		if !factory.OnboardingConfigValue().CodexSubscription {
			return nil, source.Diagnostic("agent.credentials.source", "Configure a ChatGPT subscription for this factory.")
		}
		connections, err := models.ListEligibleCodexConnections(tx, factory.OrganizationID)
		if err != nil {
			return nil, err
		}
		if len(connections) == 0 {
			return nil, source.Diagnostic("agent.credentials.source", "Connect a ChatGPT subscription for a member of this factory.")
		}
		return agent, nil
	}
	component := agent.component()
	if component == models.SuperPlaneRunnerComponent && override == "" {
		_, err := models.ResolveRunnableHostedLLMModel(tx, factory.OrganizationID, &factory.ID)
		if errors.Is(err, models.ErrSuperPlaneRunnerNoModel) || errors.Is(err, models.ErrSuperPlaneRunnerModelNotAllowed) {
			return nil, source.Diagnostic("agent.harness", "Enable a SuperPlane model for this factory or select another agent.")
		}
		return agent, err
	}

	config := map[string]any{"credentials": agent.credentials(), "model": agent.model()}
	agent.applyLLMProvider(config)
	if override != "" {
		allowed, err := models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, component, config, override)
		if err != nil {
			return nil, err
		}
		if !allowed {
			if component == models.SuperPlaneRunnerComponent {
				return nil, source.Diagnostic("agent.model", "Select a hosted model available in the workspace. Use provider::model format.")
			}
			return nil, source.Diagnostic("agent.model", "Select a model available for this connection in the factory.")
		}
		agent.Model = override
		return agent, nil
	}
	model, err := models.ResolveBYOKRunnerModel(tx, factory.OrganizationID, &factory.ID, component, config)
	if errors.Is(err, models.ErrSelectableLLMModelNotAllowed) {
		config["model"] = ""
		model, err = models.ResolveBYOKRunnerModel(tx, factory.OrganizationID, &factory.ID, component, config)
	}
	if errors.Is(err, models.ErrSelectableLLMModelNotAllowed) {
		return nil, source.Diagnostic("agent.model", "Enable a model for this agent in the factory settings.")
	}
	agent.Model = model
	return agent, err
}

func findAutomationAgentConnection(tx *gorm.DB, factory *models.Factory, component, name string) (*intakeAgent, error) {
	connections, err := models.ListIntegrations(tx, factory.OrganizationID)
	if err != nil {
		return nil, err
	}
	var agent *intakeAgent
	for i := range connections {
		connection := &connections[i]
		if connection.InstallationName != name {
			continue
		}
		candidate := intakeAgentFromIntegration(connection)
		if candidate == nil || candidate.Component != component {
			continue
		}
		if agent != nil {
			return nil, nil
		}
		agent = candidate
	}
	return fillCustomProviderModel(tx, factory, agent), nil
}

func automationSubscriptionScheduleOwner(tx *gorm.DB, factory *models.Factory, source *automationspec.Document) (string, error) {
	for index, trigger := range source.Triggers {
		if trigger.Use != automationspec.Scheduled {
			continue
		}
		owner, err := resolveAutomationSubscriptionScheduleOwner(tx, factory)
		if _, message, ok := grpcerrors.HandlerStatus(err); ok {
			return "", source.Diagnostic(fmt.Sprintf("triggers[%d].use", index), message)
		}
		return owner, err
	}
	return "", nil
}

func resolveAutomationSubscriptionScheduleOwner(tx *gorm.DB, factory *models.Factory) (string, error) {
	owner, err := uuid.Parse(factory.OnboardingConfigValue().CodexDefaultOwnerID)
	if err != nil || owner == uuid.Nil {
		return "", grpcerrors.FailedPrecondition(nil, "Select a default owner with a connected ChatGPT subscription for scheduled runs.")
	}
	eligible, err := models.SubscriptionOwnerEligible(tx, factory.OrganizationID, factory.ID, owner)
	if err != nil {
		return "", err
	}
	if !eligible {
		return "", grpcerrors.FailedPrecondition(nil, "Reconnect the default owner's ChatGPT subscription to use scheduled runs.")
	}
	return owner.String(), nil
}
