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
	err := actions.ValidateUUIDs(req.DomainId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// For now, only support organization groups as the interface only has AddUserToGroup(orgID, userID, group)
	// TODO: Update authorization service interface to support domain types
	if req.DomainType != pb.DomainType_DOMAIN_TYPE_ORGANIZATION {
		return nil, status.Error(codes.Unimplemented, "only organization groups are currently supported")
	}

	err = authService.AddUserToGroup(req.DomainId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to add user to group")
	}

	return &pb.AddUserToGroupResponse{}, nil
}

func RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest, authService authorization.Authorization) (*pb.RemoveUserFromGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// For now, only support organization groups as the interface only has RemoveUserFromGroup(orgID, userID, group)
	// TODO: Update authorization service interface to support domain types
	if req.DomainType != pb.DomainType_DOMAIN_TYPE_ORGANIZATION {
		return nil, status.Error(codes.Unimplemented, "only organization groups are currently supported")
	}

	err = authService.RemoveUserFromGroup(req.DomainId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pb.RemoveUserFromGroupResponse{}, nil
}

func ListGroups(ctx context.Context, req *pb.ListGroupsRequest, authService authorization.Authorization) (*pb.ListGroupsResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// For now, only support organization groups as the interface only has GetGroups(orgID)
	// TODO: Update authorization service interface to support domain types
	if req.DomainType != pb.DomainType_DOMAIN_TYPE_ORGANIZATION {
		return nil, status.Error(codes.Unimplemented, "only organization groups are currently supported")
	}

	groupNames, err := authService.GetGroups(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get groups")
	}

	// Convert group names to Group objects
	groups := make([]*pb.Group, len(groupNames))
	for i, groupName := range groupNames {
		// Get the role for this group - for now we'll leave it empty as the interface doesn't provide it
		// TODO: Update authorization service to return group details including roles
		groups[i] = &pb.Group{
			Name:       groupName,
			DomainType: req.DomainType,
			DomainId:   req.DomainId,
			Role:       "", // TODO: get actual role from service
		}
	}

	return &pb.ListGroupsResponse{
		Groups: groups,
	}, nil
}

func GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest, authService authorization.Authorization) (*pb.GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// For now, only support organization groups as the interface only has GetGroupUsers(orgID, group)
	// TODO: Update authorization service interface to support domain types
	if req.DomainType != pb.DomainType_DOMAIN_TYPE_ORGANIZATION {
		return nil, status.Error(codes.Unimplemented, "only organization groups are currently supported")
	}

	userIDs, err := authService.GetGroupUsers(req.DomainId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	// Create group object for response
	group := &pb.Group{
		Name:       req.GroupName,
		DomainType: req.DomainType,
		DomainId:   req.DomainId,
		Role:       "", // TODO: get actual role from service
	}

	return &pb.GetGroupUsersResponse{
		UserIds: userIDs,
		Group:   group,
	}, nil
}
