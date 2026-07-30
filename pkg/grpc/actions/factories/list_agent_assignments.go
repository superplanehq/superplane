package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListAgentAssignments(ctx context.Context, organizationID string, req *pb.ListAgentAssignmentsRequest) (*pb.ListAgentAssignmentsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	agentID, err := parseAgentID(req.GetAgentId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	tx := database.DB(ctx)
	if _, err := models.FindFactoryAgent(tx, orgID, factoryID, agentID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	assignments, err := models.ListFactoryAgentAssignmentsForAgent(tx, orgID, factoryID, agentID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list agent assignments")
	}

	return &pb.ListAgentAssignmentsResponse{
		Assignments: serializeAgentAssignments(assignments),
	}, nil
}
