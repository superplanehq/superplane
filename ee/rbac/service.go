// Package rbac implements the Enterprise custom role and group operations.
// It is licensed under the SuperPlane Enterprise license in ee/LICENSE.
package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/enterprise"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
)

// Service checks the organization entitlement before each operation, so a
// request that bypasses the HTTP gateway cannot create or expand Enterprise
// access without a grant.
type Service struct {
	authService  authorization.Authorization
	entitlements licensing.Entitlements
}

var _ enterprise.Rbac = (*Service)(nil)

func NewService(authService authorization.Authorization, entitlements licensing.Entitlements) *Service {
	return &Service{
		authService:  authService,
		entitlements: entitlements,
	}
}

func (s *Service) CreateRole(ctx context.Context, domainType, domainID string, role *pbRoles.Role) (*pbRoles.CreateRoleResponse, error) {
	if err := s.require(ctx, organizationID(domainType, domainID), licensing.FeatureCustomRoles); err != nil {
		return nil, err
	}

	return createRole(ctx, domainType, domainID, role, s.authService)
}

func (s *Service) UpdateRole(ctx context.Context, domainType, domainID, roleName string, spec *pbRoles.Role_Spec) (*pbRoles.UpdateRoleResponse, error) {
	if err := s.require(ctx, organizationID(domainType, domainID), licensing.FeatureCustomRoles); err != nil {
		return nil, err
	}

	return updateRole(ctx, domainType, domainID, roleName, spec, s.authService)
}

func (s *Service) CreateGroup(ctx context.Context, domainType, domainID string, group *pbGroups.Group) (*pbGroups.CreateGroupResponse, error) {
	if err := s.requireGroupRole(ctx, domainType, domainID, group.GetSpec().GetRole()); err != nil {
		return nil, err
	}

	return createGroup(ctx, domainType, domainID, group, s.authService)
}

func (s *Service) UpdateGroup(ctx context.Context, domainType, domainID, groupName string, spec *pbGroups.Group_Spec) (*pbGroups.UpdateGroupResponse, error) {
	if err := s.requireGroupRole(ctx, domainType, domainID, spec.GetRole()); err != nil {
		return nil, err
	}

	return updateGroup(ctx, domainType, domainID, groupName, spec, s.authService)
}

func (s *Service) AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string) (*pbGroups.AddUserToGroupResponse, error) {
	parsed, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.PermissionDenied(licensing.ErrNotLicensed, licensing.ErrNotLicensed.Error())
	}
	if err := s.require(ctx, parsed, licensing.FeatureGroups); err != nil {
		return nil, err
	}

	return addUserToGroup(ctx, orgID, domainType, domainID, userID, userEmail, groupName, s.authService)
}

// requireGroupRole also requires custom roles when a group write assigns a
// custom role, because the group members gain that role.
func (s *Service) requireGroupRole(ctx context.Context, domainType, domainID, role string) error {
	orgID := organizationID(domainType, domainID)
	if err := s.require(ctx, orgID, licensing.FeatureGroups); err != nil {
		return err
	}

	if role == "" || s.authService.IsDefaultRole(role, domainType) {
		return nil
	}

	return s.require(ctx, orgID, licensing.FeatureCustomRoles)
}

func (s *Service) require(ctx context.Context, orgID uuid.UUID, feature licensing.Feature) error {
	if err := licensing.Require(ctx, s.entitlements, orgID, feature); err != nil {
		return grpcerrors.PermissionDenied(err, err.Error())
	}

	return nil
}

// organizationID resolves the organization that owns the domain. Only
// organization domains carry the organization ID directly. Any other domain
// cannot be resolved here, so the caller denies the operation.
func organizationID(domainType, domainID string) uuid.UUID {
	if domainType != models.DomainTypeOrganization {
		return uuid.Nil
	}

	id, err := uuid.Parse(domainID)
	if err != nil {
		return uuid.Nil
	}

	return id
}
