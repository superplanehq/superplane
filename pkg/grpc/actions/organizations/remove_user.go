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
		// Serialize with invitation acceptance, which takes the same lock.
		if _, err := models.LockOrganization(tx, orgUUID); err != nil {
			return err
		}

		user, err := models.FindActiveUserByIDInTransaction(tx, orgID, userID)
		if err != nil {
			return err
		}

		if user.IsOwner {
			if err := models.RefuseIfLastOrganizationOwner(tx, orgUUID, user.ID); err != nil {
				return err
			}
		}

		// Operator invitations grant no member management, so they stay valid.
		if err := models.CancelPendingOrganizationInvitationsByInviter(tx, orgUUID, user.ID, models.RoleOrgOperator); err != nil {
			return err
		}

		// Group memberships are separate from direct roles. Both must go,
		// or a later invitation restores the old authority.
		if err := authService.RemoveUserFromOrganizationInTransaction(tx, user.ID.String(), orgID); err != nil {
			log.Errorf("Error removing organization grants for %s: %v", user.ID.String(), err)
			return grpcerrors.Internal(err, "error removing role")
		}

		if err := user.SoftDelete(tx, time.Now(), ""); err != nil {
			return err
		}

		capacity, err = billing.RecheckSeatCapacity(tx, orgUUID, "removal")
		return err
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

	billing.CapacityAlerts{capacity}.Emit()
	// The freed seat stays paid and reusable. Removal never changes the
	// Polar seat count, so it needs no billing service.
	return &pb.RemoveUserResponse{}, nil
}
