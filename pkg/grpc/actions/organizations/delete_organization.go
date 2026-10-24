package organizations

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

const liveSubscriptionDeleteReason = "ORGANIZATION_SUBSCRIPTION_LIVE"

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := organization.SoftDelete(tx); err != nil {
			return err
		}
		return authService.DestroyOrganization(tx, organization.ID.String())
	})
	if errors.Is(err, models.ErrOrganizationSubscriptionLive) {
		return nil, grpcerrors.FailedPreconditionWithReason(models.ErrOrganizationSubscriptionLive.Error(), liveSubscriptionDeleteReason)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}
	if err != nil {
		log.Errorf("Error deleting organization %s: %v", orgID, err)
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
