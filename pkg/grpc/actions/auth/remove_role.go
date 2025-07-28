package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveRole(ctx context.Context, domainType string, domainID string, roleName, userID, userEmail string, authService authorization.Authorization) (*pb.RemoveRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	userId, err := ResolveUserIDWithoutCreation(userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID or Email")
	}

	err = authService.RemoveRole(userId, roleName, domainID, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove role")
	}

	return &pb.RemoveRoleResponse{}, nil
}
