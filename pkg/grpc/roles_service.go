package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

type RoleService struct {
	pb.UnimplementedRolesServer
	authService authorization.Authorization
}

func NewRoleService(authService authorization.Authorization) *RoleService {
	return &RoleService{
		authService: authService,
	}
}

func (s *RoleService) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	return auth.AssignRole(ctx, req, s.authService)
}

func (s *RoleService) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.RemoveRoleResponse, error) {
	return auth.RemoveRole(ctx, req, s.authService)
}

func (s *RoleService) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return auth.ListRoles(ctx, req, s.authService)
}

func (s *RoleService) DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest) (*pb.DescribeRoleResponse, error) {
	return auth.DescribeRole(ctx, req, s.authService)
}

func (s *RoleService) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	return auth.CreateRole(ctx, req, s.authService)
}

func (s *RoleService) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	return auth.UpdateRole(ctx, req, s.authService)
}

func (s *RoleService) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	return auth.DeleteRole(ctx, req, s.authService)
}
