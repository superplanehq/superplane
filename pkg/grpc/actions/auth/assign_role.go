package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AssignRole(ctx context.Context, orgID, domainType, domainID, roleName, userID, userEmail string, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	requesterID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}

	if user.ID.String() == requesterID {
		return nil, status.Error(codes.PermissionDenied, "cannot change your own role")
	}

	if user.IsServiceAccount() && roleName == models.RoleOrgOwner {
		return nil, status.Error(codes.InvalidArgument, "service accounts cannot be assigned the org_owner role")
	}

	err = authService.AssignRole(user.ID.String(), roleName, domainID, domainType)
	if err != nil {
		log.Errorf("Error assigning role %s to %s: %v", roleName, user.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
