package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateAgentAssignment(ctx context.Context, organizationID string, req *pb.CreateAgentAssignmentRequest) (*pb.CreateAgentAssignmentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	agentID, err := parseAgentID(req.GetAgentId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	orderIDs, err := parseOrderIDs(req.GetOrderIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	tx := database.DB(ctx)
	if _, err := models.FindFactoryAgent(tx, orgID, factoryID, agentID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	for _, orderID := range orderIDs {
		if _, err := models.FindFactoryWorkOrder(tx, orgID, factoryID, orderID); err != nil {
			return nil, factoryErrorToStatus(err, "failed to create agent assignment")
		}
	}

	params := make([]models.CreateFactoryAgentAssignmentParams, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		params = append(params, models.CreateFactoryAgentAssignmentParams{
			OrganizationID: orgID,
			FactoryID:      factoryID,
			AgentID:        agentID,
			WorkOrderID:    orderID,
			Instructions:   req.GetInstructions(),
		})
	}

	assignments, err := models.CreateFactoryAgentAssignments(tx, params)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create agent assignment")
	}

	return &pb.CreateAgentAssignmentResponse{
		Assignments: serializeAgentAssignments(assignments),
	}, nil
}
