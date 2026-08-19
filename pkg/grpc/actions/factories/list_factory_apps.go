package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactoryApps(ctx context.Context, organizationID string, req *pb.ListFactoryAppsRequest) (*pb.ListFactoryAppsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory apps")
	}

	return &pb.ListFactoryAppsResponse{
		Apps: serializeFactoryApps(canvases),
	}, nil
}
