package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeAgentAssignment(ctx context.Context, organizationID string, req *pb.DescribeAgentAssignmentRequest) (*pb.DescribeAgentAssignmentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe agent assignment")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe agent assignment")
	}

	agentID, err := parseAgentID(req.GetAgentId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe agent assignment")
	}

	assignmentID, err := parseAssignmentID(req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe agent assignment")
	}

	assignment, err := models.FindFactoryAgentAssignment(database.DB(ctx), orgID, factoryID, agentID, assignmentID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe agent assignment")
	}

	return &pb.DescribeAgentAssignmentResponse{
		Assignment: serializeAgentAssignment(assignment),
	}, nil
}
