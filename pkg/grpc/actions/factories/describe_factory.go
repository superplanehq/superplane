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

	factory, err := models.FindFactory(database.DB(ctx), orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	tx := database.DB(ctx)
	lines, err := models.ListFactoryLines(tx, orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	return &pb.DescribeFactoryResponse{
		Factory: serializeFactoryWithLines(factory, lines),
	}, nil
}
