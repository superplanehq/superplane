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

func UninstallApplication(ctx context.Context, orgID string, ID string) (*pb.UninstallApplicationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	installationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid installation ID: %v", err)
	}

	appInstallation, err := models.FindAppInstallation(org, installationID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "application installation not found: %v", err)
	}

	//
	// NOTE: this performs a soft deletion of the app installation.
	// The reason for a soft deletion here is to ensure we deprovision
	// and delete its webhooks before we delete the app installation itself.
	//
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		webhooks, err := models.ListAppInstallationWebhooks(tx, appInstallation.ID)
		if err != nil {
			return status.Error(codes.Internal, "failed to list application installation webhooks")
		}

		//
		// We soft-delete all the webhooks associated with the installation as well.
		//
		for _, webhook := range webhooks {
			err = tx.Delete(&webhook).Error
			if err != nil {
				return status.Error(codes.Internal, "failed to delete webhook")
			}
		}

		err = appInstallation.SoftDeleteInTransaction(tx)
		if err != nil {
			return status.Error(codes.Internal, "failed to delete application installation")
		}

		return nil
	})

	return &pb.UninstallApplicationResponse{}, nil
}
