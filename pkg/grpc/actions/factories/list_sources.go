package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListSources(ctx context.Context, organizationID string, req *pb.ListSourcesRequest) (*pb.ListSourcesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory sources")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory sources")
	}

	tx := database.DB(ctx)
	if _, err := loadFactory(tx, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory sources")
	}

	sources, err := models.ListFactorySources(tx, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory sources")
	}

	protoSources, err := serializeFactorySources(tx, orgID, sources)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory sources")
	}

	return &pb.ListSourcesResponse{
		Sources: protoSources,
	}, nil
}
