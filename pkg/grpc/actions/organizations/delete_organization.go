package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	err = database.TransactionWithContext(ctx, database.DefaultOrganizationMutationTimeout, "DeleteOrganization", func(tx *gorm.DB) error {
		if txErr := models.SoftDeleteOrganizationInTransaction(tx, organization.ID.String()); txErr != nil {
			log.Errorf("Error deleting organization %s: %v", orgID, txErr)
			return txErr
		}

		if txErr := authService.DestroyOrganization(tx, organization.ID.String()); txErr != nil {
			log.Errorf("Error deleting organization roles for %s: %v", orgID, txErr)
			return txErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Infof(
		"Organization %s (%s) soft-deleted by user %s",
		organization.Name,
		organization.ID.String(),
		userID,
	)

	return &pb.DeleteOrganizationResponse{}, nil
}
