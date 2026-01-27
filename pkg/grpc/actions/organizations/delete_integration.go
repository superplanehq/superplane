package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteIntegration(ctx context.Context, orgID string, ID string) (*pb.DeleteIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	integrationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid integration ID: %v", err)
	}

	integration, err := models.FindAppInstallation(org, integrationID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "integration not found: %v", err)
	}

	//
	// NOTE: this performs a soft deletion of the integration.
	// The reason for a soft deletion here is to ensure we deprovision
	// and delete its webhooks before we delete the integration itself.
	//
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		webhooks, err := models.ListAppInstallationWebhooks(tx, integration.ID)
		if err != nil {
			return status.Error(codes.Internal, "failed to list integration webhooks")
		}

		//
		// We soft-delete all the webhooks associated with the integration as well.
		//
		for _, webhook := range webhooks {
			err = tx.Delete(&webhook).Error
			if err != nil {
				return status.Error(codes.Internal, "failed to delete webhook")
			}
		}

		err = integration.SoftDeleteInTransaction(tx)
		if err != nil {
			return status.Error(codes.Internal, "failed to delete integration")
		}

		return nil
	})

	return &pb.DeleteIntegrationResponse{}, nil
}
