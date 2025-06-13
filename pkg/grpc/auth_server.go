package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
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

func (s *AuthorizationServer) CheckPermission(ctx context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return actions.CheckPermission(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListUserPermissions(ctx context.Context, req *pb.ListUserPermissionsRequest) (*pb.ListUserPermissionsResponse, error) {
	return actions.ListUserPermissions(ctx, req, s.authService)
}

func (s *AuthorizationServer) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	return actions.AssignRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.RemoveRoleResponse, error) {
	return actions.RemoveRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return actions.ListRoles(ctx, req, s.authService)
}

func (s *AuthorizationServer) DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest) (*pb.DescribeRoleResponse, error) {
	return actions.DescribeRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListAccessibleOrganizations(ctx context.Context, req *pb.ListAccessibleOrganizationsRequest) (*pb.ListAccessibleOrganizationsResponse, error) {
	return actions.ListAccessibleOrganizations(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListAccessibleCanvases(ctx context.Context, req *pb.ListAccessibleCanvasesRequest) (*pb.ListAccessibleCanvasesResponse, error) {
	return actions.ListAccessibleCanvases(ctx, req, s.authService)
}

func (s *AuthorizationServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return actions.GetUserRoles(ctx, req, s.authService)
}

func (s *AuthorizationServer) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	return actions.CreateGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	return actions.AddUserToGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	return actions.RemoveUserFromGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest) (*pb.ListOrganizationGroupsResponse, error) {
	return actions.ListOrganizationGroups(ctx, req, s.authService)
}

func (s *AuthorizationServer) GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest) (*pb.GetGroupUsersResponse, error) {
	return actions.GetGroupUsers(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListOrganizationUsersForRole(ctx context.Context, req *pb.ListOrganizationUsersForRoleRequest) (*pb.ListOrganizationUsersForRoleResponse, error) {
	return actions.ListOrganizationUsersForRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) ListCanvasUsersForRole(ctx context.Context, req *pb.ListCanvasUsersForRoleRequest) (*pb.ListCanvasUsersForRoleResponse, error) {
	return actions.ListCanvasUsersForRole(ctx, req, s.authService)
}
