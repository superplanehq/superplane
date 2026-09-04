package organizations

import (
	"context"
	"slices"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID, userID string) (*pb.RemoveUserResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	ownerIDs, err := authService.GetOrgUsersForRole(ctx, models.RoleOrgOwner, orgID)
	if err != nil {
		log.Errorf("Error determining owners for org %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "error determining organization owners")
	}

	if len(ownerIDs) <= 1 && slices.Contains(ownerIDs, user.ID.String()) {
		return nil, grpcerrors.FailedPrecondition(nil, "cannot remove the last organization owner")
	}

	roles, err := authService.GetUserRolesForOrg(ctx, user.ID.String(), orgID)
	if err != nil {
		log.Errorf("Error determining user roles for %s: %v", user.ID.String(), err)
		return nil, grpcerrors.Internal(err, "error determining user roles")
	}

	// Delete the user and remove roles atomically. The user soft-delete and
	// role removal are performed together so a partial failure cannot leave the
	// user in an inconsistent state (e.g. roles removed but still active).
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Unscoped().
			Model(user).
			Update("deleted_at", now).
			Update("updated_at", now).
			Update("token_hash", nil).
			Error; err != nil {
			return err
		}

		for _, role := range roles{
			if err := authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization); err != nil {
				return err 
			}
		}

		return nil
	})
	if err != nil {
		log.Errorf("Error removing user %s: %v", user.ID.String(), orgID, err)
		return nil, grpcerrors.Internal(err, "error removing user")
	}

	return &pb.RemoveUserResponse{}, nil
}
