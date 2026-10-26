package organizations

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

const (
	liveSubscriptionDeleteReason       = "ORGANIZATION_SUBSCRIPTION_LIVE"
	pendingCheckoutDeleteReason        = "ORGANIZATION_CHECKOUT_PENDING"
	unverifiedSubscriptionDeleteReason = "ORGANIZATION_SUBSCRIPTION_UNVERIFIED"
	billingChangedDeleteMessage        = "The billing record changed during deletion. Try again."
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

	// Ask Polar before the transaction: the request must not hold the
	// organization lock. SoftDelete checks the clearance again under the lock.
	clearance, err := polar.VerifyOrganizationDeletion(ctx, organization.ID)
	if err != nil {
		return nil, deleteOrganizationError(orgID, err)
	}

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := organization.SoftDelete(tx, clearance); err != nil {
			return err
		}
		return authService.DestroyOrganization(tx, organization.ID.String())
	})
	if err != nil {
		return nil, deleteOrganizationError(orgID, err)
	}

	log.Infof(
		"Organization %s (%s) soft-deleted by user %s",
		organization.Name,
		organization.ID.String(),
		userID,
	)

	return &pb.DeleteOrganizationResponse{}, nil
}

func deleteOrganizationError(orgID string, err error) error {
	switch {
	case errors.Is(err, models.ErrOrganizationSubscriptionLive):
		log.WithError(err).WithField("organization_id", orgID).Info("organization deletion blocked by a live subscription")
		return grpcerrors.FailedPreconditionWithReason(models.ErrOrganizationSubscriptionLive.Error(), liveSubscriptionDeleteReason)
	case errors.Is(err, models.ErrOrganizationCheckoutPending):
		log.WithError(err).WithField("organization_id", orgID).Info("organization deletion blocked by a pending checkout")
		return grpcerrors.FailedPreconditionWithReason(models.ErrOrganizationCheckoutPending.Error(), pendingCheckoutDeleteReason)
	case errors.Is(err, models.ErrOrganizationSubscriptionUnverified):
		log.WithError(err).WithField("organization_id", orgID).Warn("organization deletion could not verify subscription termination")
		return grpcerrors.UnavailableWithReason(models.ErrOrganizationSubscriptionUnverified.Error(), unverifiedSubscriptionDeleteReason)
	case errors.Is(err, models.ErrBillingPlanChanged):
		log.WithField("organization_id", orgID).Info("organization deletion stopped by a billing change")
		return grpcerrors.Conflict(err, billingChangedDeleteMessage)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "organization not found")
	default:
		log.Errorf("Error deleting organization %s: %v", orgID, err)
		return err
	}
}
