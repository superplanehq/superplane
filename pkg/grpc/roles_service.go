package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RoleService struct {
	pb.UnimplementedRolesServer
	authService authorization.Authorization
}

func NewRoleService(authService authorization.Authorization) *RoleService {
	return &RoleService{
		authService: authService,
	}
}

func (s *RoleService) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	orgID := authorization.OrganizationIDFromContext(ctx)
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)
	return auth.AssignRole(ctx, orgID, domainType, domainID, req.RoleName, req.UserId, req.UserEmail, s.authService)
}

func (s *RoleService) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)
	return auth.ListRoles(ctx, domainType, domainID, s.authService)
}

func (s *RoleService) DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest) (*pb.DescribeRoleResponse, error) {
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)
	return auth.DescribeRole(ctx, domainType, domainID, req.RoleName, s.authService)
}

func (s *RoleService) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)
	return auth.CreateRole(ctx, domainType, domainID, req.Role, s.authService)
}

func (s *RoleService) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)

	if req.Role == nil {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	return auth.UpdateRole(ctx, domainType, domainID, req.RoleName, req.Role.Spec, s.authService)
}

func (s *RoleService) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	domainType := authorization.DomainTypeFromContext(ctx)
	domainID := authorization.DomainIDFromContext(ctx)
	return auth.DeleteRole(ctx, domainType, domainID, req.RoleName, s.authService)
}
