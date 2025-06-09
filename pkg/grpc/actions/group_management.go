package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest, authService authorization.AuthorizationServiceInterface) (*pb.AddUserToGroupResponse, error) {
	// Validate UUIDs
	err := ValidateUUIDs(req.OrgId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	// Validate required fields
	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	// Add user to group
	err = authService.AddUserToGroup(req.OrgId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add user to group")
	}

	return &pb.AddUserToGroupResponse{}, nil
}

func RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest, authService authorization.AuthorizationServiceInterface) (*pb.RemoveUserFromGroupResponse, error) {
	// Validate UUIDs
	err := ValidateUUIDs(req.OrgId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	// Validate required fields
	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	// Remove user from group
	err = authService.RemoveUserFromGroup(req.OrgId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pb.RemoveUserFromGroupResponse{}, nil
}

func ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest, authService authorization.AuthorizationServiceInterface) (*pb.ListOrganizationGroupsResponse, error) {
	// Validate UUID
	err := ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	// Get groups
	groups, err := authService.GetGroups(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get organization groups")
	}

	return &pb.ListOrganizationGroupsResponse{
		Groups: groups,
	}, nil
}

func GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest, authService authorization.AuthorizationServiceInterface) (*pb.GetGroupUsersResponse, error) {
	// Validate UUID
	err := ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	// Validate required fields
	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	// Get group users
	userIDs, err := authService.GetGroupUsers(req.OrgId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	return &pb.GetGroupUsersResponse{
		UserIds: userIDs,
	}, nil
}
