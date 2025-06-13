package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest, authService authorization.Authorization) (*pb.AddUserToGroupResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	err = authService.AddUserToGroup(req.OrgId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add user to group")
	}

	return &pb.AddUserToGroupResponse{}, nil
}

func RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest, authService authorization.Authorization) (*pb.RemoveUserFromGroupResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	err = authService.RemoveUserFromGroup(req.OrgId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pb.RemoveUserFromGroupResponse{}, nil
}

func ListOrganizationGroups(ctx context.Context, req *pb.ListOrganizationGroupsRequest, authService authorization.Authorization) (*pb.ListOrganizationGroupsResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	groups, err := authService.GetGroups(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get organization groups")
	}

	return &pb.ListOrganizationGroupsResponse{
		Groups: groups,
	}, nil
}

func GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest, authService authorization.Authorization) (*pb.GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	userIDs, err := authService.GetGroupUsers(req.OrgId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	return &pb.GetGroupUsersResponse{
		UserIds: userIDs,
	}, nil
}
