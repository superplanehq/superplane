package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
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

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, id)
		if err != nil {
			return err
		}

		if err := factory.SoftDeleteCanvases(tx); err != nil {
			return err
		}

		return factory.SoftDelete(tx)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	return &pb.DeleteFactoryResponse{}, nil
}
