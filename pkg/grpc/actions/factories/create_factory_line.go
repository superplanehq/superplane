package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func CreateFactoryLine(ctx context.Context, organizationID string, req *pb.CreateFactoryLineRequest) (*pb.CreateFactoryLineResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to create factory line")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}
	factoryID := factory.ID

	steps, err := parseLineSteps(db, orgID, factoryID, req.GetSteps())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	line, err := factory.CreateLine(db, name, steps)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	return &pb.CreateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
