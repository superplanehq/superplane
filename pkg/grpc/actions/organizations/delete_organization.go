package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
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
	actorID, _ := authentication.GetAccountIDFromMetadata(ctx)
	if actorID == "" {
		user, findErr := models.FindActiveUserByIDInTransaction(database.DB(ctx), orgID, userID)
		if findErr == nil && user.AccountID != nil {
			actorID = user.AccountID.String()
		}
	}

	// Ask Polar before the transaction: the request must not hold the
	// organization lock. SoftDelete checks the clearance again under the lock.
	clearance, err := billing.PrepareDeletion(ctx, organization.ID)
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
	distinctID := actorID
	actorType := productanalytics.ActorUser
	properties := productanalytics.Properties{
		"deletion_source": "direct",
		"deleted_at":      organization.DeletedAt.Time.UTC().Format(time.RFC3339Nano),
	}
	if actorID == "" {
		distinctID = productanalytics.SyntheticDistinctID(organization.ID.String(), productanalytics.ActorSystem, "organization")
		actorType = productanalytics.ActorSystem
	} else {
		properties["actor_account_id"] = actorID
	}
	productanalytics.Capture(productanalytics.Event{
		Name:       productanalytics.EventOrganizationDeleted,
		EventID:    productanalytics.StableEventID(productanalytics.EventOrganizationDeleted, organization.ID.String()),
		DistinctID: distinctID, OrganizationID: organization.ID.String(), ActorType: actorType,
		Source:     productanalytics.SourceFromContext(ctx, productanalytics.SourceAPI),
		Properties: properties,
	})

	return &pb.DeleteOrganizationResponse{}, nil
}

func deleteOrganizationError(orgID string, err error) error {
	switch {
	case errors.Is(err, billing.ErrOrganizationSubscriptionLive):
		log.WithError(err).WithField("organization_id", orgID).Info("organization deletion blocked by a live subscription")
		return grpcerrors.FailedPreconditionWithReason(billing.ErrOrganizationSubscriptionLive.Error(), liveSubscriptionDeleteReason)
	case errors.Is(err, billing.ErrOrganizationCheckoutPending):
		log.WithError(err).WithField("organization_id", orgID).Info("organization deletion blocked by a pending checkout")
		return grpcerrors.FailedPreconditionWithReason(billing.ErrOrganizationCheckoutPending.Error(), pendingCheckoutDeleteReason)
	case errors.Is(err, billing.ErrOrganizationSubscriptionUnverified):
		log.WithError(err).WithField("organization_id", orgID).Warn("organization deletion could not verify subscription termination")
		return grpcerrors.UnavailableWithReason(billing.ErrOrganizationSubscriptionUnverified.Error(), unverifiedSubscriptionDeleteReason)
	case errors.Is(err, billing.ErrBillingPlanChanged):
		log.WithField("organization_id", orgID).Info("organization deletion stopped by a billing change")
		return grpcerrors.Conflict(err, billingChangedDeleteMessage)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "organization not found")
	default:
		log.Errorf("Error deleting organization %s: %v", orgID, err)
		return err
	}
}
