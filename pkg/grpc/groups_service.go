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
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.CreateGroup(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.AddUserToGroup(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.RemoveUserFromGroup(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) ListGroups(ctx context.Context, req *pb.ListGroupsRequest) (*pb.ListGroupsResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.ListGroups(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) GetGroup(ctx context.Context, req *pb.GetGroupRequest) (*pb.GetGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.GetGroup(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest) (*pb.GetGroupUsersResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.GetGroupUsers(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.UpdateGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.UpdateGroup(ctx, domainType, domainID, req, s.authService)
}

func (s *GroupsService) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DeleteGroup(ctx, domainType, domainID, req, s.authService)
}
