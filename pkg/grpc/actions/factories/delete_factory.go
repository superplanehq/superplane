package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DeleteFactory(ctx context.Context, organizationID, factoryID string) (*pb.DeleteFactoryResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory")
	}

	var factory *models.Factory
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		factory, err = findFactory(tx, orgID, factoryID)
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

	captureFactoryProductEvent(ctx, database.DB(ctx), factory, uuid.Nil, productanalytics.EventFactoryDeleted,
		productanalytics.StableEventID(productanalytics.EventFactoryDeleted, factory.ID.String()), nil)
	return &pb.DeleteFactoryResponse{}, nil
}
