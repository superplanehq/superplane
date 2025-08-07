package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AssignRole(ctx context.Context, domainType, domainID, roleName, userID, userEmail string, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)

	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}

	err = authService.AssignRole(user.ID.String(), roleName, domainID, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
