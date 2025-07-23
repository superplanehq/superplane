package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
)

type GroupsService struct {
	pb.UnimplementedGroupsServer
	authService authorization.Authorization
}

func NewGroupsService(authService authorization.Authorization) *GroupsService {
	return &GroupsService{
		authService: authService,
	}
}

func (s *GroupsService) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	return auth.CreateGroup(ctx, req, s.authService)
}

func (s *GroupsService) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	return auth.AddUserToGroup(ctx, req, s.authService)
}

func (s *GroupsService) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	return auth.RemoveUserFromGroup(ctx, req, s.authService)
}

func (s *GroupsService) ListGroups(ctx context.Context, req *pb.ListGroupsRequest) (*pb.ListGroupsResponse, error) {
	return auth.ListGroups(ctx, req, s.authService)
}

func (s *GroupsService) GetGroup(ctx context.Context, req *pb.GetGroupRequest) (*pb.GetGroupResponse, error) {
	return auth.GetGroup(ctx, req, s.authService)
}

func (s *GroupsService) GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest) (*pb.GetGroupUsersResponse, error) {
	return auth.GetGroupUsers(ctx, req, s.authService)
}

func (s *GroupsService) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.UpdateGroupResponse, error) {
	return auth.UpdateGroup(ctx, req, s.authService)
}

func (s *GroupsService) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	return auth.DeleteGroup(ctx, req, s.authService)
}
