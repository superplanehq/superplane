package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GroupsService struct {
	pb.UnimplementedGroupsServer
	authService authorization.Authorization
	rbac        enterprise.Rbac
}

func NewGroupsService(authService authorization.Authorization, registry *enterprise.Registry) (*GroupsService, error) {
	rbac, err := enterprise.Get[enterprise.Rbac](registry, enterprise.RBAC)
	if err != nil {
		return nil, err
	}

	return &GroupsService{
		authService: authService,
		rbac:        rbac,
	}, nil
}

func (s *GroupsService) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.rbac.CreateGroup(ctx, domainType, domainID, req.Group)
}

func (s *GroupsService) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*pb.AddUserToGroupResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.rbac.AddUserToGroup(ctx, orgID, domainType, domainID, req.UserId, req.UserEmail, req.GroupName)
}

func (s *GroupsService) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*pb.RemoveUserFromGroupResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.RemoveUserFromGroup(ctx, orgID, domainType, domainID, req.UserId, req.UserEmail, req.GroupName, s.authService)
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

	return s.rbac.UpdateGroup(ctx, domainType, domainID, req.GroupName, req.Group.Spec)
}

func (s *GroupsService) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DeleteGroup(ctx, domainType, domainID, req.GroupName, s.authService)
}
