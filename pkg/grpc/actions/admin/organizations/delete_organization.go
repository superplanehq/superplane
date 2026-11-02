package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/organizations"
	"gorm.io/gorm"
)

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	accountID, ok := authentication.GetAccountIDFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "Account authentication is required.")
	}
	admin, err := models.FindAccountByID(accountID)
	if err != nil || !admin.IsInstallationAdmin() {
		return nil, grpcerrors.PermissionDenied(nil, "Installation admin access is required.")
	}
	orgID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}
	db := database.DB(ctx)
	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, deletionError(err)
	}

	// Provider verification must finish before taking the organization lock.
	// SoftDelete validates the billing snapshot again under that lock.
	clearance, err := billing.PrepareDeletion(ctx, orgID)
	if err != nil {
		return nil, deletionError(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := organization.SoftDelete(tx, clearance); err != nil {
			return err
		}
		return authService.DestroyOrganization(tx, orgID.String())
	})
	if err != nil {
		return nil, deletionError(err)
	}

	productanalytics.Capture(productanalytics.Event{
		Name:       productanalytics.EventOrganizationDeleted,
		EventID:    productanalytics.StableEventID(productanalytics.EventOrganizationDeleted, orgID.String()),
		DistinctID: admin.ID.String(), OrganizationID: orgID.String(), ActorType: productanalytics.ActorUser,
		Source: productanalytics.SourceFromContext(ctx, productanalytics.SourceUI),
		Properties: productanalytics.Properties{
			"actor_account_id": admin.ID.String(), "deletion_source": "admin",
			"deleted_at": organization.DeletedAt.Time.UTC().Format(time.RFC3339Nano),
		},
	})
	log.WithFields(log.Fields{
		"admin_account_id": admin.ID.String(), "organization_id": orgID.String(),
		"action": "delete_organization",
	}).Info("organization deleted")
	return &pb.DeleteOrganizationResponse{}, nil
}

func deletionError(err error) error {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "Organization not found.")
	case errors.Is(err, billing.ErrOrganizationSubscriptionLive):
		return grpcerrors.Conflict(err, billing.ErrOrganizationSubscriptionLive.Error())
	case errors.Is(err, billing.ErrOrganizationCheckoutPending):
		return grpcerrors.Conflict(err, billing.ErrOrganizationCheckoutPending.Error())
	case errors.Is(err, billing.ErrOrganizationSubscriptionUnverified):
		return grpcerrors.UnavailableWithReason(billing.ErrOrganizationSubscriptionUnverified.Error(), "ORGANIZATION_SUBSCRIPTION_UNVERIFIED")
	case errors.Is(err, billing.ErrBillingPlanChanged):
		return grpcerrors.Conflict(err, "The billing record changed during deletion. Try again.")
	default:
		return grpcerrors.Internal(err, "Could not delete the organization. Try again.")
	}
}
