package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
)

type AuthorizationServer struct {
	pb.UnimplementedAuthorizationServer
	authService authorization.Authorization
}

func NewAuthorizationServer(authService authorization.Authorization) *AuthorizationServer {
	return &AuthorizationServer{
		authService: authService,
	}
}

func (s *AuthorizationServer) ListUserPermissions(ctx context.Context, req *pb.ListUserPermissionsRequest) (*pb.ListUserPermissionsResponse, error) {
	return auth.ListUserPermissions(ctx, req, s.authService)
}

func (s *AuthorizationServer) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	return auth.AssignRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.RemoveRoleResponse, error) {
	return auth.RemoveRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return auth.ListRoles(ctx, req, s.authService)
}

func (s *AuthorizationServer) DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest) (*pb.DescribeRoleResponse, error) {
	return auth.DescribeRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return auth.GetUserRoles(ctx, req, s.authService)
}

func (s *AuthorizationServer) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	return auth.CreateGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	return auth.AddUserToGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	return auth.RemoveUserFromGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest) (*pb.ListOrganizationGroupsResponse, error) {
	return auth.ListOrganizationGroups(ctx, req, s.authService)
}

func (s *AuthorizationServer) GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest) (*pb.GetGroupUsersResponse, error) {
	return auth.GetGroupUsers(ctx, req, s.authService)
}
