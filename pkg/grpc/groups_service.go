package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	return auth.CreateGroup(ctx, domainType, domainID, req.Group, s.authService)
}

func (s *GroupsService) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.AddUserToGroup(ctx, domainType, domainID, req.UserId, req.UserEmail, req.GroupName, s.authService)
}

func (s *GroupsService) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.RemoveUserFromGroup(ctx, domainType, domainID, req.UserId, req.UserEmail, req.GroupName, s.authService)
}

func (s *GroupsService) ListGroups(ctx context.Context, req *pb.ListGroupsRequest) (*pb.ListGroupsResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.ListGroups(ctx, domainType, domainID, s.authService)
}

func (s *GroupsService) DescribeGroup(ctx context.Context, req *pb.DescribeGroupRequest) (*pb.DescribeGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DescribeGroup(ctx, domainType, domainID, req.GroupName, s.authService)
}

func (s *GroupsService) ListGroupUsers(ctx context.Context, req *pb.ListGroupUsersRequest) (*pb.ListGroupUsersResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.ListGroupUsers(ctx, domainType, domainID, req.GroupName, s.authService)
}

func (s *GroupsService) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.UpdateGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)

	if req.Group == nil {
		return nil, status.Error(codes.InvalidArgument, "group must be specified")
	}

	return auth.UpdateGroup(ctx, domainType, domainID, req.GroupName, req.Group.Spec, s.authService)
}

func (s *GroupsService) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DeleteGroup(ctx, domainType, domainID, req.GroupName, s.authService)
}
