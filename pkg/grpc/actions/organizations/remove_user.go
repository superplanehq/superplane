package organizations

import (
	"context"
	"slices"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID, userID string) (*pb.RemoveUserResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	ownerIDs, err := authService.GetOrgUsersForRole(models.RoleOrgOwner, orgID)
	if err != nil {
		log.Errorf("Error determining owners for org %s: %v", orgID, err)
		return nil, status.Error(codes.Internal, "error determining organization owners")
	}

	if len(ownerIDs) <= 1 && slices.Contains(ownerIDs, user.ID.String()) {
		return nil, status.Error(codes.FailedPrecondition, "cannot remove the last organization owner")
	}

	roles, err := authService.GetUserRolesForOrg(user.ID.String(), orgID)
	if err != nil {
		log.Errorf("Error determining user roles for %s: %v", user.ID.String(), err)
		return nil, status.Error(codes.Internal, "error determining user roles")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, role := range roles {
			if err := authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization); err != nil {
				log.Errorf("Error removing role %s for %s: %v", role.Name, user.ID.String(), err)
				return err
			}
		}

		return user.DeleteInTransaction(tx)
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "error removing user")
	}

	return &pb.RemoveUserResponse{}, nil
}
