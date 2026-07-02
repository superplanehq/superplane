package actions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const listIntegrationsActionName = "list_integrations"

type listIntegrationsAction struct{}

func (listIntegrationsAction) Name() string {
	return listIntegrationsActionName
}

func (listIntegrationsAction) Execute(ctx context.Context, session agents.AgentSessionContext, _ Input) (any, error) {
	orgID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return integrationsResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	integrations, err := listConnectedIntegrations(ctx, orgID)
	if err != nil {
		return integrationsResult{}, err
	}

	return integrationsResult{
		Action:       "list_integrations",
		CanvasID:     session.CanvasID,
		Integrations: integrations,
	}, nil
}

func listConnectedIntegrations(ctx context.Context, orgID uuid.UUID) ([]integrationResult, error) {
	db := database.DB(ctx)
	integrations, err := models.ListIntegrations(db, orgID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}

	result := make([]integrationResult, 0, len(integrations))
	for _, integration := range integrations {
		result = append(result, integrationResult{
			ID:     integration.ID.String(),
			Name:   integration.InstallationName,
			Vendor: integration.AppName,
			State:  integration.State,
		})
	}
	return result, nil
}
