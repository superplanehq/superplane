package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID, userID string) (*pb.RemoveUserResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization ID")
	}

	var capacity billing.CapacityAlert
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		capacity, txErr = RemoveUserInTransaction(tx, authService, orgUUID, userID)
		return txErr
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "user not found")
		}
		if errors.Is(err, models.ErrLastOrganizationOwner) {
			return nil, grpcerrors.FailedPrecondition(err, models.ErrLastOrganizationOwner.Error())
		}
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}

		return nil, grpcerrors.Internal(err, "error deleting user")
	}

	if err := messages.PublishOrganizationCodexConnectionUpdated(database.DB(ctx), orgUUID); err != nil {
		log.WithError(err).Warn("Failed to publish Codex member removal")
	}
	billing.CapacityAlerts{capacity}.Emit()
	// The freed seat stays paid and reusable. Removal never changes the
	// Polar seat count, so it needs no billing service.
	return &pb.RemoveUserResponse{}, nil
}

// RemoveUserInTransaction removes one member on the caller's transaction.
// The caller publishes the Codex update and emits the capacity alert after commit.
func RemoveUserInTransaction(tx *gorm.DB, authService authorization.Authorization, orgUUID uuid.UUID, userID string) (billing.CapacityAlert, error) {
	// Serialize with invitation acceptance, which takes the same lock.
	if _, err := models.LockOrganization(tx, orgUUID); err != nil {
		return nil, err
	}

	user, err := models.FindActiveUserByIDInTransaction(tx, orgUUID.String(), userID)
	if err != nil {
		return nil, err
	}

	if user.IsOwner {
		if err := models.RefuseIfLastOrganizationOwner(tx, orgUUID, user.ID); err != nil {
			return nil, err
		}
	}

	// Operator invitations grant no member management, so they stay valid.
	if err := models.CancelPendingOrganizationInvitationsByInviter(tx, orgUUID, user.ID, models.RoleOrgOperator); err != nil {
		return nil, err
	}

	// Group memberships are separate from direct roles. Both must go,
	// or a later invitation restores the old authority.
	if err := authService.RemoveUserFromOrganizationInTransaction(tx, user.ID.String(), orgUUID.String()); err != nil {
		log.Errorf("Error removing organization grants for %s: %v", user.ID.String(), err)
		return nil, grpcerrors.Internal(err, "error removing role")
	}

	if err := user.SoftDelete(tx, time.Now(), ""); err != nil {
		return nil, err
	}

	return billing.RecheckSeatCapacity(tx, orgUUID, "removal")
}
