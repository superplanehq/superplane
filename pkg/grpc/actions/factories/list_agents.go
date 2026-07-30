package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListAgents(ctx context.Context, organizationID string, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory agents")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory agents")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory agents")
	}

	agents, err := models.ListFactoryAgents(tx, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory agents")
	}

	return &pb.ListAgentsResponse{
		Agents: serializeFactoryAgents(agents),
	}, nil
}
