package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	"github.com/superplanehq/superplane/pkg/licensing"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RoleService struct {
	pb.UnimplementedRolesServer
	authService   authorization.Authorization
	entitlements  licensing.Entitlements
	accessControl EnterpriseAccessControl
}

func NewRoleService(
	authService authorization.Authorization,
	entitlements licensing.Entitlements,
	accessControl EnterpriseAccessControl,
) *RoleService {
	return &RoleService{
		authService:   authService,
		entitlements:  entitlements,
		accessControl: accessControl,
	}
}

func (s *RoleService) AssignRole(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)

	// Built-in roles are Community features. Assigning a custom role grants
	// Enterprise access, so it requires the custom roles entitlement.
	if !s.authService.IsDefaultRole(req.RoleName, domainType) && !licensing.Allows(s.entitlements, licensing.FeatureCustomRoles) {
		return nil, notLicensed()
	}

	return auth.AssignRole(ctx, orgID, domainType, domainID, req.RoleName, req.UserId, req.UserEmail, s.authService)
}

func (s *RoleService) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.ListRoles(ctx, domainType, domainID, s.authService)
}

func (s *RoleService) DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest) (*pb.DescribeRoleResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DescribeRole(ctx, domainType, domainID, req.RoleName, s.authService)
}

func (s *RoleService) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return s.accessControl.CreateRole(ctx, domainType, domainID, req.Role)
}

func (s *RoleService) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)

	if req.Role == nil {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	return s.accessControl.UpdateRole(ctx, domainType, domainID, req.RoleName, req.Role.Spec)
}

func (s *RoleService) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return auth.DeleteRole(ctx, domainType, domainID, req.RoleName, s.authService)
}
