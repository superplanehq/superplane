package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func AssignRole(ctx context.Context, orgID, domainType, domainID, roleName, userID, userEmail string, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	requesterID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if roleName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "invalid role")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "user not found")
	}

	if user.ID.String() == requesterID {
		return nil, grpcerrors.PermissionDenied(nil, "cannot change your own role")
	}

	if user.IsServiceAccount() && roleName == models.RoleOrgOwner {
		return nil, grpcerrors.InvalidArgument(nil, "service accounts cannot be assigned the org_owner role")
	}

	err = authService.AssignRole(user.ID.String(), roleName, domainID, domainType)
	if err != nil {
		log.Errorf("Error assigning role %s to %s: %v", roleName, user.ID.String(), err)
		return nil, grpcerrors.Internal(err, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
