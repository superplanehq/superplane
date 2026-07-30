package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateAgent(ctx context.Context, organizationID string, req *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory agent")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory agent")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to create factory agent")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory agent")
	}

	agent, err := models.CreateFactoryAgent(tx, orgID, factoryID, name, req.GetDescription(), models.FactoryAgentSpec{})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory agent")
	}

	return &pb.CreateAgentResponse{
		Agent: serializeFactoryAgent(agent),
	}, nil
}
