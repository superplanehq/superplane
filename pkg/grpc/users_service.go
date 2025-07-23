package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
)

type UsersService struct {
	pb.UnimplementedUsersServer
	authService authorization.Authorization
}

func NewUsersService(authService authorization.Authorization) *UsersService {
	return &UsersService{
		authService: authService,
	}
}

func (s *UsersService) ListUserPermissions(ctx context.Context, req *pb.ListUserPermissionsRequest) (*pb.ListUserPermissionsResponse, error) {
	return auth.ListUserPermissions(ctx, req, s.authService)
}

func (s *UsersService) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return auth.GetUserRoles(ctx, req, s.authService)
}

func (s *UsersService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	return auth.ListUsers(ctx, req, s.authService)
}
