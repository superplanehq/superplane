package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactoryAutomations(ctx context.Context, organizationID string, req *pb.ListFactoryAutomationsRequest) (*pb.ListFactoryAutomationsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	return &pb.ListFactoryAutomationsResponse{
		Automations: serializeFactoryAutomations(canvases),
	}, nil
}
