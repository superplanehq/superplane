package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func UpdateFactory(ctx context.Context, organizationID string, req *pb.UpdateFactoryRequest) (*pb.UpdateFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	id, err := parseFactoryID(req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	if err := factory.Update(db, req.Name, req.Description, req.Key); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory")
	}

	return &pb.UpdateFactoryResponse{
		Factory: serializeFactoryWithLines(factory, lines),
	}, nil
}
