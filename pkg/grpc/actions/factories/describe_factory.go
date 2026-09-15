package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DescribeFactory(ctx context.Context, organizationID, factoryID string) (*pb.DescribeFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory")
	}

	return &pb.DescribeFactoryResponse{
		Factory: serialized,
	}, nil
}
