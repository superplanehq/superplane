package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUser(ctx context.Context, authService authorization.Authorization, orgID, userID string) (*pb.RemoveUserResponse, error) {
	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	//
	// TODO: this should all be inside of a transaction
	// Remove organization roles
	//
	roles, err := authService.GetUserRolesForOrg(user.ID.String(), orgID)
	if err != nil {
		log.Errorf("Error determing user roles for %s: %v", user.ID.String(), err)
		return nil, status.Error(codes.Internal, "error determing user roles")
	}

	for _, role := range roles {
		err = authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization)
		if err != nil {
			log.Errorf("Error removing role %s for %s: %v", role.Name, user.ID.String(), err)
			return nil, status.Error(codes.Internal, "error removing role")
		}
	}

	err = user.Delete()
	if err != nil {
		return nil, status.Error(codes.Internal, "error deleting user")
	}

	return &pb.RemoveUserResponse{}, nil
}
