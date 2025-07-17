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
	genericReq.DomainID = req.OrganizationId

	genericResp, err := auth.CreateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToCreateOrganizationGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) AddUserToOrganizationGroup(ctx context.Context, req *pb.AddUserToOrganizationGroupRequest) (*pb.AddUserToOrganizationGroupResponse, error) {
	genericReq := auth.ConvertAddUserToOrganizationGroupRequest(req)
	genericReq.DomainID = req.OrganizationId

	err := auth.AddUserToGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.AddUserToOrganizationGroupResponse{}, nil
}

func (s *AuthorizationServer) RemoveUserFromOrganizationGroup(ctx context.Context, req *pb.RemoveUserFromOrganizationGroupRequest) (*pb.RemoveUserFromOrganizationGroupResponse, error) {
	genericReq := auth.ConvertRemoveUserFromOrganizationGroupRequest(req)
	genericReq.DomainID = req.OrganizationId

	err := auth.RemoveUserFromGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveUserFromOrganizationGroupResponse{}, nil
}

func (s *AuthorizationServer) ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest) (*pb.ListOrganizationGroupsResponse, error) {
	genericReq := auth.ConvertListOrganizationGroupsRequest(req)
	genericReq.DomainID = req.OrganizationId

	genericResp, err := auth.ListGroups(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToListOrganizationGroupsResponse(genericResp.Groups), nil
}

func (s *AuthorizationServer) GetOrganizationGroup(ctx context.Context, req *pb.GetOrganizationGroupRequest) (*pb.GetOrganizationGroupResponse, error) {
	genericReq := auth.ConvertGetOrganizationGroupRequest(req)
	genericReq.DomainID = req.OrganizationId

	genericResp, err := auth.GetGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToGetOrganizationGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) GetOrganizationGroupUsers(ctx context.Context, req *pb.GetOrganizationGroupUsersRequest) (*pb.GetOrganizationGroupUsersResponse, error) {
	genericReq := auth.ConvertGetOrganizationGroupUsersRequest(req)
	genericReq.DomainID = req.OrganizationId

	genericResp, err := auth.GetGroupUsers(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToGetOrganizationGroupUsersResponse(genericResp), nil
}

func (s *AuthorizationServer) GetOrganizationUsers(ctx context.Context, req *pb.GetOrganizationUsersRequest) (*pb.GetOrganizationUsersResponse, error) {
	return auth.GetOrganizationUsers(ctx, req, s.authService)
}

func (s *AuthorizationServer) CreateCanvasGroup(ctx context.Context, req *pb.CreateCanvasGroupRequest) (*pb.CreateCanvasGroupResponse, error) {
	genericReq, err := auth.ConvertCreateCanvasGroupRequest(req)
	if err != nil {
		return nil, err
	}

	genericResp, err := auth.CreateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToCreateCanvasGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) AddUserToCanvasGroup(ctx context.Context, req *pb.AddUserToCanvasGroupRequest) (*pb.AddUserToCanvasGroupResponse, error) {
	genericReq, err := auth.ConvertAddUserToCanvasGroupRequest(req)
	if err != nil {
		return nil, err
	}

	err = auth.AddUserToGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.AddUserToCanvasGroupResponse{}, nil
}

func (s *AuthorizationServer) RemoveUserFromCanvasGroup(ctx context.Context, req *pb.RemoveUserFromCanvasGroupRequest) (*pb.RemoveUserFromCanvasGroupResponse, error) {
	genericReq, err := auth.ConvertRemoveUserFromCanvasGroupRequest(req)
	if err != nil {
		return nil, err
	}

	err = auth.RemoveUserFromGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveUserFromCanvasGroupResponse{}, nil
}

func (s *AuthorizationServer) ListCanvasGroups(ctx context.Context, req *pb.ListCanvasGroupsRequest) (*pb.ListCanvasGroupsResponse, error) {
	genericReq, err := auth.ConvertListCanvasGroupsRequest(req)
	if err != nil {
		return nil, err
	}

	genericResp, err := auth.ListGroups(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToListCanvasGroupsResponse(genericResp.Groups), nil
}

func (s *AuthorizationServer) GetCanvasGroup(ctx context.Context, req *pb.GetCanvasGroupRequest) (*pb.GetCanvasGroupResponse, error) {
	genericReq, err := auth.ConvertGetCanvasGroupRequest(req)
	if err != nil {
		return nil, err
	}

	genericResp, err := auth.GetGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToGetCanvasGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) GetCanvasGroupUsers(ctx context.Context, req *pb.GetCanvasGroupUsersRequest) (*pb.GetCanvasGroupUsersResponse, error) {
	genericReq, err := auth.ConvertGetCanvasGroupUsersRequest(req)
	if err != nil {
		return nil, err
	}

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

func (s *AuthorizationServer) UpdateOrganizationGroup(ctx context.Context, req *pb.UpdateOrganizationGroupRequest) (*pb.UpdateOrganizationGroupResponse, error) {
	genericReq := auth.ConvertUpdateOrganizationGroupRequest(req)
	genericReq.DomainID = req.OrganizationId

	genericResp, err := auth.UpdateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToUpdateOrganizationGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) UpdateCanvasGroup(ctx context.Context, req *pb.UpdateCanvasGroupRequest) (*pb.UpdateCanvasGroupResponse, error) {
	genericReq, err := auth.ConvertUpdateCanvasGroupRequest(req)
	if err != nil {
		return nil, err
	}

	genericResp, err := auth.UpdateGroup(ctx, genericReq, s.authService)
	if err != nil {
		return nil, err
	}

	return auth.ConvertToUpdateCanvasGroupResponse(genericResp), nil
}

func (s *AuthorizationServer) DeleteOrganizationGroup(ctx context.Context, req *pb.DeleteOrganizationGroupRequest) (*pb.DeleteOrganizationGroupResponse, error) {
	return auth.DeleteOrganizationGroup(ctx, req, s.authService)
}

func (s *AuthorizationServer) DeleteCanvasGroup(ctx context.Context, req *pb.DeleteCanvasGroupRequest) (*pb.DeleteCanvasGroupResponse, error) {
	return auth.DeleteCanvasGroup(ctx, req, s.authService)
}
