package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

const (
	liveSubscriptionDeleteMessage = "Cancel the subscription in Billing before you delete the organization. You can delete it after the paid period ends."
	liveSubscriptionDeleteReason  = "ORGANIZATION_SUBSCRIPTION_LIVE"
)

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	hasLiveSubscription, err := models.OrganizationHasLivePolarSubscription(database.Conn(), organization.ID)
	if err != nil {
		log.Errorf("Error checking the subscription of organization %s before deletion: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to check the organization subscription")
	}
	if hasLiveSubscription {
		return nil, grpcerrors.FailedPreconditionWithReason(liveSubscriptionDeleteMessage, liveSubscriptionDeleteReason)
	}

	tx := database.Conn().Begin()
	err = models.SoftDeleteOrganizationInTransaction(tx, organization.ID.String())
	if err != nil {
		tx.Rollback()
		log.Errorf("Error deleting organization %s: %v", orgID, err)
		return nil, err
	}

	err = authService.DestroyOrganization(tx, organization.ID.String())
	if err != nil {
		tx.Rollback()
		log.Errorf("Error deleting organization roles for %s: %v", orgID, err)
		return nil, err
	}

	err = tx.Commit().Error
	if err != nil {
		log.Errorf("Error committing transaction for organization %s (%s) deletion: %v", organization.Name, organization.ID.String(), err)
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
