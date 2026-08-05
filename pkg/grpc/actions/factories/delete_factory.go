package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func DeleteFactory(ctx context.Context, organizationID, factoryID string) (*pb.DeleteFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	id, err := parseFactoryID(factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	if err := factory.SoftDelete(db); err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	return &pb.DeleteFactoryResponse{}, nil
}
