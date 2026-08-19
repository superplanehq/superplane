package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeFactory(ctx context.Context, organizationID, factoryID string) (*pb.DescribeFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	id, err := parseFactoryID(factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	return &pb.DescribeFactoryResponse{
		Factory: serializeFactoryWithLines(factory, lines),
	}, nil
}
