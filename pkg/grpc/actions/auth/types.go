package auth

import (
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GroupRequest struct {
	DomainId   string
	GroupName  string
	DomainType pb.DomainType
}

type GroupUserRequest struct {
	DomainId   string
	GroupName  string
	DomainType pb.DomainType
	UserId     string
}

type CreateGroupRequest struct {
	DomainId   string
	GroupName  string
	DomainType pb.DomainType
	Role       string
}

type CreateGroupResponse struct {
	Group *pb.Group
}

type ListGroupsResponse struct {
	Groups []*pb.Group
}

type GetGroupUsersRequest struct {
	DomainId   string
	GroupName  string
	DomainType pb.DomainType
}

type GetGroupUsersResponse struct {
	UserIds []string
	Group   *pb.Group
}

func ValidateGroupRequest(req *GroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ValidateGroupUserRequest(req *GroupUserRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ValidateCreateGroupRequest(req *CreateGroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.Role == "" {
		return status.Error(codes.InvalidArgument, "role must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	return nil
}

func ConvertDomainType(domainType pb.DomainType) (string, error) {
	switch domainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return "org", nil
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		return "canvas", nil
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported domain type")
	}
}

// Organization group adapters
func ConvertCreateOrganizationGroupRequest(req *pb.CreateOrganizationGroupRequest) *CreateGroupRequest {
	return &CreateGroupRequest{
		DomainId:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
		Role:       req.Role,
	}
}

func ConvertToCreateOrganizationGroupResponse(resp *CreateGroupResponse) *pb.CreateOrganizationGroupResponse {
	return &pb.CreateOrganizationGroupResponse{
		Group: resp.Group,
	}
}

func ConvertAddUserToOrganizationGroupRequest(req *pb.AddUserToOrganizationGroupRequest) *GroupUserRequest {
	return &GroupUserRequest{
		DomainId:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
		UserId:     req.UserId,
	}
}

func ConvertRemoveUserFromOrganizationGroupRequest(req *pb.RemoveUserFromOrganizationGroupRequest) *GroupUserRequest {
	return &GroupUserRequest{
		DomainId:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
		UserId:     req.UserId,
	}
}

func ConvertListOrganizationGroupsRequest(req *pb.ListOrganizationGroupsRequest) *GroupRequest {
	return &GroupRequest{
		DomainId:   "", // Organization ID will be set by server
		GroupName:  "", // Not needed for list
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
}

func ConvertGetOrganizationGroupUsersRequest(req *pb.GetOrganizationGroupUsersRequest) *GetGroupUsersRequest {
	return &GetGroupUsersRequest{
		DomainId:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
}

func ConvertToGetOrganizationGroupUsersResponse(resp *GetGroupUsersResponse) *pb.GetOrganizationGroupUsersResponse {
	return &pb.GetOrganizationGroupUsersResponse{
		UserIds: resp.UserIds,
		Group:   resp.Group,
	}
}

func ConvertToListOrganizationGroupsResponse(groups []*pb.Group) *pb.ListOrganizationGroupsResponse {
	return &pb.ListOrganizationGroupsResponse{
		Groups: groups,
	}
}

// Canvas group adapters
func ConvertCreateCanvasGroupRequest(req *pb.CreateCanvasGroupRequest) *CreateGroupRequest {
	return &CreateGroupRequest{
		DomainId:   req.CanvasId,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
		Role:       req.Role,
	}
}

func ConvertToCreateCanvasGroupResponse(resp *CreateGroupResponse) *pb.CreateCanvasGroupResponse {
	return &pb.CreateCanvasGroupResponse{
		Group: resp.Group,
	}
}

func ConvertAddUserToCanvasGroupRequest(req *pb.AddUserToCanvasGroupRequest) *GroupUserRequest {
	return &GroupUserRequest{
		DomainId:   req.CanvasId,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
		UserId:     req.UserId,
	}
}

func ConvertRemoveUserFromCanvasGroupRequest(req *pb.RemoveUserFromCanvasGroupRequest) *GroupUserRequest {
	return &GroupUserRequest{
		DomainId:   req.CanvasId,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
		UserId:     req.UserId,
	}
}

func ConvertListCanvasGroupsRequest(req *pb.ListCanvasGroupsRequest) *GroupRequest {
	return &GroupRequest{
		DomainId:   req.CanvasId,
		GroupName:  "", // Not needed for list
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}
}

func ConvertGetCanvasGroupUsersRequest(req *pb.GetCanvasGroupUsersRequest) *GetGroupUsersRequest {
	return &GetGroupUsersRequest{
		DomainId:   req.CanvasId,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}
}

func ConvertToGetCanvasGroupUsersResponse(resp *GetGroupUsersResponse) *pb.GetCanvasGroupUsersResponse {
	return &pb.GetCanvasGroupUsersResponse{
		UserIds: resp.UserIds,
		Group:   resp.Group,
	}
}

func ConvertToListCanvasGroupsResponse(groups []*pb.Group) *pb.ListCanvasGroupsResponse {
	return &pb.ListCanvasGroupsResponse{
		Groups: groups,
	}
}