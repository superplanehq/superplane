package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func ListFactories(ctx context.Context, organizationID string) (*pb.ListFactoriesResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factories")
	}

	factories, err := models.ListFactories(database.DB(ctx), orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factories")
	}

	return &pb.ListFactoriesResponse{
		Factories: serializeFactories(factories),
	}, nil
}
