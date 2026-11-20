package factories

import (
	"errors"
	"strings"

	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func resolveAutomationAgent(tx *gorm.DB, factory *models.Factory, source *automationspec.Document) (*intakeAgent, error) {
	agent := &intakeAgent{Component: models.SuperPlaneRunnerComponent}
	if source.Agent != nil {
		agent.Model = strings.TrimSpace(source.Agent.Model)
	}
	if agent.Model == "" {
		_, err := models.ResolveRunnableHostedLLMModel(tx, factory.OrganizationID, &factory.ID)
		if errors.Is(err, models.ErrSuperPlaneRunnerNoModel) || errors.Is(err, models.ErrSuperPlaneRunnerModelNotAllowed) {
			return nil, source.Diagnostic("agent", "Enable a SuperPlane-hosted model for this factory.")
		}
		return agent, err
	}

	allowed, err := models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, agent.Component, nil, agent.Model)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, source.Diagnostic("agent.model", "Select a hosted model available in the workspace. Use provider::model format.")
	}
	return agent, nil
}
