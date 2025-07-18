package auth

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GroupRequest struct {
	DomainID   string
	GroupName  string
	DomainType pb.DomainType
}

type GroupUserRequest struct {
	DomainID   string
	GroupName  string
	DomainType pb.DomainType
	UserID     string
	UserEmail  string
}

type CreateGroupRequest struct {
	DomainID    string
	GroupName   string
	DomainType  pb.DomainType
	Role        string
	DisplayName string
	Description string
}

type CreateGroupResponse struct {
	Group *pb.Group
}

type ListGroupsResponse struct {
	Groups []*pb.Group
}

type GetGroupUsersRequest struct {
	DomainID   string
	GroupName  string
	DomainType pb.DomainType
}

type GetGroupUsersResponse struct {
	Users []*pb.User
	Group *pb.Group
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

func ConvertCanvasIdOrNameToId(canvasIdOrName string) (string, error) {
	if _, err := uuid.Parse(canvasIdOrName); err != nil {
		canvas, err := models.FindCanvasByName(canvasIdOrName)
		if err != nil {
			return "", status.Error(codes.NotFound, "canvas not found")
		}
		return canvas.ID.String(), nil
	}
	return canvasIdOrName, nil
}

func ConvertDomainType(domainType pb.DomainType) (string, error) {
	switch domainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return authorization.DomainOrg, nil
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		return authorization.DomainCanvas, nil
	default:
		return "", status.Error(codes.InvalidArgument, "unsupported domain type")
	}
}

// Organization group adapters
func ConvertCreateOrganizationGroupRequest(req *pb.CreateOrganizationGroupRequest) *CreateGroupRequest {
	return &CreateGroupRequest{
		DomainID:    "", // Organization ID will be set by server
		GroupName:   req.GroupName,
		DomainType:  pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
		Role:        req.Role,
		DisplayName: req.DisplayName,
		Description: req.Description,
	}
}

func ConvertToCreateOrganizationGroupResponse(resp *CreateGroupResponse) *pb.CreateOrganizationGroupResponse {
	return &pb.CreateOrganizationGroupResponse{
		Group: resp.Group,
	}
}

func ConvertAddUserToOrganizationGroupRequest(req *pb.AddUserToOrganizationGroupRequest) *GroupUserRequest {
	groupReq := &GroupUserRequest{
		DomainID:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
	
	// Handle oneof user_identifier
	switch identifier := req.UserIdentifier.(type) {
	case *pb.AddUserToOrganizationGroupRequest_UserId:
		groupReq.UserID = identifier.UserId
	case *pb.AddUserToOrganizationGroupRequest_UserEmail:
		groupReq.UserEmail = identifier.UserEmail
	}
	
	return groupReq
}

func ConvertRemoveUserFromOrganizationGroupRequest(req *pb.RemoveUserFromOrganizationGroupRequest) *GroupUserRequest {
	groupReq := &GroupUserRequest{
		DomainID:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
	
	// Handle oneof user_identifier
	switch identifier := req.UserIdentifier.(type) {
	case *pb.RemoveUserFromOrganizationGroupRequest_UserId:
		groupReq.UserID = identifier.UserId
	case *pb.RemoveUserFromOrganizationGroupRequest_UserEmail:
		groupReq.UserEmail = identifier.UserEmail
	}
	
	return groupReq
}

func ConvertListOrganizationGroupsRequest(req *pb.ListOrganizationGroupsRequest) *GroupRequest {
	return &GroupRequest{
		DomainID:   "", // Organization ID will be set by server
		GroupName:  "", // Not needed for list
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
}

func ConvertGetOrganizationGroupUsersRequest(req *pb.GetOrganizationGroupUsersRequest) *GetGroupUsersRequest {
	return &GetGroupUsersRequest{
		DomainID:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
}

func ConvertToGetOrganizationGroupUsersResponse(resp *GetGroupUsersResponse) *pb.GetOrganizationGroupUsersResponse {
	return &pb.GetOrganizationGroupUsersResponse{
		Users: resp.Users,
		Group: resp.Group,
	}
}

func ConvertToListOrganizationGroupsResponse(groups []*pb.Group) *pb.ListOrganizationGroupsResponse {
	return &pb.ListOrganizationGroupsResponse{
		Groups: groups,
	}
}

func ConvertGetOrganizationGroupRequest(req *pb.GetOrganizationGroupRequest) *GetGroupRequest {
	return &GetGroupRequest{
		DomainID:   "", // Organization ID will be set by server
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
	}
}

func ConvertToGetOrganizationGroupResponse(resp *GetGroupResponse) *pb.GetOrganizationGroupResponse {
	return &pb.GetOrganizationGroupResponse{
		Group: resp.Group,
	}
}

func ConvertUpdateOrganizationGroupRequest(req *pb.UpdateOrganizationGroupRequest) *UpdateGroupRequest {
	return &UpdateGroupRequest{
		DomainID:    "", // Organization ID will be set by server
		GroupName:   req.GroupName,
		DomainType:  pb.DomainType_DOMAIN_TYPE_ORGANIZATION,
		Role:        req.Role,
		DisplayName: req.DisplayName,
		Description: req.Description,
	}
}

func ConvertToUpdateOrganizationGroupResponse(resp *UpdateGroupResponse) *pb.UpdateOrganizationGroupResponse {
	return &pb.UpdateOrganizationGroupResponse{
		Group: resp.Group,
	}
}

// Canvas group adapters
func ConvertCreateCanvasGroupRequest(req *pb.CreateCanvasGroupRequest) (*CreateGroupRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	return &CreateGroupRequest{
		DomainID:    canvasID,
		GroupName:   req.GroupName,
		DomainType:  pb.DomainType_DOMAIN_TYPE_CANVAS,
		Role:        req.Role,
		DisplayName: req.DisplayName,
		Description: req.Description,
	}, nil
}

func ConvertToCreateCanvasGroupResponse(resp *CreateGroupResponse) *pb.CreateCanvasGroupResponse {
	return &pb.CreateCanvasGroupResponse{
		Group: resp.Group,
	}
}

func ConvertAddUserToCanvasGroupRequest(req *pb.AddUserToCanvasGroupRequest) (*GroupUserRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	
	groupReq := &GroupUserRequest{
		DomainID:   canvasID,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}
	
	// Handle oneof user_identifier
	switch identifier := req.UserIdentifier.(type) {
	case *pb.AddUserToCanvasGroupRequest_UserId:
		groupReq.UserID = identifier.UserId
	case *pb.AddUserToCanvasGroupRequest_UserEmail:
		groupReq.UserEmail = identifier.UserEmail
	}
	
	return groupReq, nil
}

func ConvertRemoveUserFromCanvasGroupRequest(req *pb.RemoveUserFromCanvasGroupRequest) (*GroupUserRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	
	groupReq := &GroupUserRequest{
		DomainID:   canvasID,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}
	
	// Handle oneof user_identifier
	switch identifier := req.UserIdentifier.(type) {
	case *pb.RemoveUserFromCanvasGroupRequest_UserId:
		groupReq.UserID = identifier.UserId
	case *pb.RemoveUserFromCanvasGroupRequest_UserEmail:
		groupReq.UserEmail = identifier.UserEmail
	}
	
	return groupReq, nil
}

func ConvertListCanvasGroupsRequest(req *pb.ListCanvasGroupsRequest) (*GroupRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	return &GroupRequest{
		DomainID:   canvasID,
		GroupName:  "", // Not needed for list
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}, nil
}

func ConvertGetCanvasGroupUsersRequest(req *pb.GetCanvasGroupUsersRequest) (*GetGroupUsersRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	return &GetGroupUsersRequest{
		DomainID:   canvasID,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}, nil
}

func ConvertToGetCanvasGroupUsersResponse(resp *GetGroupUsersResponse) *pb.GetCanvasGroupUsersResponse {
	return &pb.GetCanvasGroupUsersResponse{
		Users: resp.Users,
		Group: resp.Group,
	}
}

func ConvertToListCanvasGroupsResponse(groups []*pb.Group) *pb.ListCanvasGroupsResponse {
	return &pb.ListCanvasGroupsResponse{
		Groups: groups,
	}
}

func ConvertGetCanvasGroupRequest(req *pb.GetCanvasGroupRequest) (*GetGroupRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	return &GetGroupRequest{
		DomainID:   canvasID,
		GroupName:  req.GroupName,
		DomainType: pb.DomainType_DOMAIN_TYPE_CANVAS,
	}, nil
}

func ConvertToGetCanvasGroupResponse(resp *GetGroupResponse) *pb.GetCanvasGroupResponse {
	return &pb.GetCanvasGroupResponse{
		Group: resp.Group,
	}
}

func ConvertUpdateCanvasGroupRequest(req *pb.UpdateCanvasGroupRequest) (*UpdateGroupRequest, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}
	return &UpdateGroupRequest{
		DomainID:    canvasID,
		GroupName:   req.GroupName,
		DomainType:  pb.DomainType_DOMAIN_TYPE_CANVAS,
		Role:        req.Role,
		DisplayName: req.DisplayName,
		Description: req.Description,
	}, nil
}

func ConvertToUpdateCanvasGroupResponse(resp *UpdateGroupResponse) *pb.UpdateCanvasGroupResponse {
	return &pb.UpdateCanvasGroupResponse{
		Group: resp.Group,
	}
}
