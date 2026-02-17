package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListUsers(ctx context.Context, domainType string, domainID string, includeServiceAccounts bool, authService authorization.Authorization) (*pb.ListUsersResponse, error) {
	users, err := GetUsersWithRolesInDomain(domainID, domainType, includeServiceAccounts, authService)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get canvas users")
	}

	return &pb.ListUsersResponse{
		Users: users,
	}, nil
}
