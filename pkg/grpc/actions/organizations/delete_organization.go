package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

const maxOrganizationDeletionAttempts = 3

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	for attempt := 0; attempt < maxOrganizationDeletionAttempts; attempt++ {
		scheduled, err := polar.CancelOrganizationSubscriptionForDeletion(ctx, database.Conn(), organization.ID)
		if err != nil {
			log.Errorf("Error canceling Business plan before deleting organization %s: %v", organization.ID.String(), err)
			return nil, grpcerrors.Internal(err, "failed to cancel the Business plan. The organization was not deleted.")
		}

		err = deleteLockedOrganization(authService, organization)
		if err == nil {
			log.Infof(
				"Organization %s (%s) soft-deleted by user %s",
				organization.Name,
				organization.ID.String(),
				userID,
			)
			return &pb.DeleteOrganizationResponse{}, nil
		}
		if errors.Is(err, polar.ErrSubscriptionChangedDuringDeletion) {
			continue
		}

		log.Errorf("Error deleting organization %s: %v", orgID, err)
		return nil, restoreSubscriptionAfterFailedDeletion(ctx, organization.ID, scheduled, err)
	}

	log.Errorf("Business plan changed during deletion of organization %s", organization.ID.String())
	return nil, grpcerrors.Internal(
		polar.ErrSubscriptionChangedDuringDeletion,
		"the Business plan changed during deletion. The organization was not deleted.",
	)
}

func deleteLockedOrganization(authService authorization.Authorization, organization *models.Organization) error {
	tx := database.Conn().Begin()
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	if _, err := models.LockOrganization(tx, organization.ID); err != nil {
		return err
	}

	changed, err := polar.SubscriptionChangedSinceDeletionCancel(tx, organization.ID)
	if err != nil {
		return err
	}
	if changed {
		return polar.ErrSubscriptionChangedDuringDeletion
	}

	if err := models.SoftDeleteOrganizationInTransaction(tx, organization.ID.String()); err != nil {
		return err
	}
	if err := authService.DestroyOrganization(tx, organization.ID.String()); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	committed = true
	return nil
}

func restoreSubscriptionAfterFailedDeletion(ctx context.Context, orgID uuid.UUID, scheduled bool, deleteErr error) error {
	if !scheduled {
		return deleteErr
	}
	if resumeErr := polar.ResumeOrganizationSubscription(ctx, database.Conn(), orgID); resumeErr != nil {
		log.Errorf("Error restoring Business plan after failed deletion of organization %s: %v", orgID.String(), resumeErr)
		return grpcerrors.Internal(resumeErr, "failed to delete the organization. The Business plan is set to end at the period end.")
	}
	return deleteErr
}
