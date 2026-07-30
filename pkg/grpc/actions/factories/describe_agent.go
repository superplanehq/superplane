package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeAgent(ctx context.Context, organizationID string, req *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory agent")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory agent")
	}

	agentID, err := parseAgentID(req.GetAgentId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory agent")
	}

	agent, err := models.FindFactoryAgent(database.DB(ctx), orgID, factoryID, agentID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory agent")
	}

	return &pb.DescribeAgentResponse{
		Agent: serializeFactoryAgent(agent),
	}, nil
}
