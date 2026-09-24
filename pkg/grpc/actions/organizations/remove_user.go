package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
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

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := models.FindActiveUserByIDInTransaction(tx, orgID, userID)
		if err != nil {
			return err
		}

		if user.IsOwner {
			if err := models.RefuseIfLastOrganizationOwner(tx, orgUUID, user.ID); err != nil {
				return err
			}
		}

		roles, err := authService.GetUserRolesForOrg(ctx, user.ID.String(), orgID)
		if err != nil {
			log.Errorf("Error determing user roles for %s: %v", user.ID.String(), err)
			return grpcerrors.Internal(err, "error determing user roles")
		}

		for _, role := range roles {
			err = authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization)
			if err != nil {
				log.Errorf("Error removing role %s for %s: %v", role.Name, user.ID.String(), err)
				return grpcerrors.Internal(err, "error removing role")
			}
		}

		return user.SoftDelete(tx, time.Now(), "")
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

	return &pb.RemoveUserResponse{}, nil
}
