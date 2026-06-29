package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func DeleteIntegration(ctx context.Context, orgID string, ID string) (*pb.DeleteIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization ID")
	}

	integrationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid integration ID")
	}

	integration, err := models.FindIntegration(org, integrationID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "integration not found")
	}

	//
	// NOTE: this performs a soft deletion of the integration.
	// The reason for a soft deletion here is to ensure we deprovision
	// and delete its webhooks before we delete the integration itself.
	//
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		webhooks, err := models.ListIntegrationWebhooks(tx, integration.ID)
		if err != nil {
			return grpcerrors.Internal(err, "failed to list integration webhooks")
		}

		//
		// We soft-delete all the webhooks associated with the integration as well.
		//
		for _, webhook := range webhooks {
			err = tx.Delete(&webhook).Error
			if err != nil {
				return grpcerrors.Internal(err, "failed to delete webhook")
			}
		}

		err = integration.SoftDeleteInTransaction(tx)
		if err != nil {
			return grpcerrors.Internal(err, "failed to delete integration")
		}

		return nil
	})

	return &pb.DeleteIntegrationResponse{}, nil
}
