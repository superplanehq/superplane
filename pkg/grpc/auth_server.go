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

func (s *AuthorizationServer) CreateOrganizationGroup(ctx context.Context, req *pb.CreateOrganizationGroupRequest) (*pb.CreateOrganizationGroupResponse, error) {
	genericReq := auth.ConvertCreateOrganizationGroupRequest(req)
	genericReq.DomainId = req.OrganizationId

	genericResp, err := auth.CreateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToCreateOrganizationGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) AddUserToOrganizationGroup(ctx context.Context, req *pb.AddUserToOrganizationGroupRequest) (*pb.AddUserToOrganizationGroupResponse, error) {
	genericReq := auth.ConvertAddUserToOrganizationGroupRequest(req)
	genericReq.DomainId = req.OrganizationId

	err := auth.AddUserToGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.AddUserToOrganizationGroupResponse{}, nil
}

func (s *AuthorizationServer) RemoveUserFromOrganizationGroup(ctx context.Context, req *pb.RemoveUserFromOrganizationGroupRequest) (*pb.RemoveUserFromOrganizationGroupResponse, error) {
	genericReq := auth.ConvertRemoveUserFromOrganizationGroupRequest(req)
	genericReq.DomainId = req.OrganizationId

	err := auth.RemoveUserFromGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveUserFromOrganizationGroupResponse{}, nil
}

func (s *AuthorizationServer) ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest) (*pb.ListOrganizationGroupsResponse, error) {
	genericReq := auth.ConvertListOrganizationGroupsRequest(req)
	genericReq.DomainId = req.OrganizationId

	genericResp, err := auth.ListGroups(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToListOrganizationGroupsResponse(genericResp.Groups), nil
}

func (s *AuthorizationServer) GetOrganizationGroupUsers(ctx context.Context, req *pb.GetOrganizationGroupUsersRequest) (*pb.GetOrganizationGroupUsersResponse, error) {
	genericReq := auth.ConvertGetOrganizationGroupUsersRequest(req)
	genericReq.DomainId = req.OrganizationId

	genericResp, err := auth.GetGroupUsers(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToGetOrganizationGroupUsersResponse(genericResp), nil
}

func (s *AuthorizationServer) CreateCanvasGroup(ctx context.Context, req *pb.CreateCanvasGroupRequest) (*pb.CreateCanvasGroupResponse, error) {
	genericReq := auth.ConvertCreateCanvasGroupRequest(req)

	genericResp, err := auth.CreateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToCreateCanvasGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) AddUserToCanvasGroup(ctx context.Context, req *pb.AddUserToCanvasGroupRequest) (*pb.AddUserToCanvasGroupResponse, error) {
	genericReq := auth.ConvertAddUserToCanvasGroupRequest(req)

	err := auth.AddUserToGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.AddUserToCanvasGroupResponse{}, nil
}

func (s *AuthorizationServer) RemoveUserFromCanvasGroup(ctx context.Context, req *pb.RemoveUserFromCanvasGroupRequest) (*pb.RemoveUserFromCanvasGroupResponse, error) {
	genericReq := auth.ConvertRemoveUserFromCanvasGroupRequest(req)

	err := auth.RemoveUserFromGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveUserFromCanvasGroupResponse{}, nil
}

func (s *AuthorizationServer) ListCanvasGroups(ctx context.Context, req *pb.ListCanvasGroupsRequest) (*pb.ListCanvasGroupsResponse, error) {
	genericReq := auth.ConvertListCanvasGroupsRequest(req)

	genericResp, err := auth.ListGroups(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToListCanvasGroupsResponse(genericResp.Groups), nil
}

func (s *AuthorizationServer) GetCanvasGroupUsers(ctx context.Context, req *pb.GetCanvasGroupUsersRequest) (*pb.GetCanvasGroupUsersResponse, error) {
	genericReq := auth.ConvertGetCanvasGroupUsersRequest(req)

	genericResp, err := auth.GetGroupUsers(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToGetCanvasGroupUsersResponse(genericResp), nil
}

func (s *AuthorizationServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	return auth.CreateRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	return auth.UpdateRole(ctx, req, s.authService)
}

func (s *AuthorizationServer) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	return auth.DeleteRole(ctx, req, s.authService)
}
