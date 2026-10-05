// Package rbac implements the Enterprise custom role and group operations.
// It is licensed under the SuperPlane Enterprise license in ee/LICENSE.
package rbac

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
)

// Service checks the installation license before each operation, so a
// request that bypasses the HTTP gateway cannot create or expand Enterprise
// access without a license.
type Service struct {
	authService  authorization.Authorization
	entitlements licensing.Entitlements
}

func NewService(authService authorization.Authorization, entitlements licensing.Entitlements) *Service {
	return &Service{
		authService:  authService,
		entitlements: entitlements,
	}
}

func (s *Service) CreateRole(ctx context.Context, domainType, domainID string, role *pbRoles.Role) (*pbRoles.CreateRoleResponse, error) {
	if err := s.require(licensing.FeatureCustomRoles); err != nil {
		return nil, err
	}

	return createRole(ctx, domainType, domainID, role, s.authService)
}

func (s *Service) UpdateRole(ctx context.Context, domainType, domainID, roleName string, spec *pbRoles.Role_Spec) (*pbRoles.UpdateRoleResponse, error) {
	if err := s.require(licensing.FeatureCustomRoles); err != nil {
		return nil, err
	}

	return updateRole(ctx, domainType, domainID, roleName, spec, s.authService)
}

func (s *Service) CreateGroup(ctx context.Context, domainType, domainID string, group *pbGroups.Group) (*pbGroups.CreateGroupResponse, error) {
	if err := s.require(licensing.FeatureGroups); err != nil {
		return nil, err
	}

	return createGroup(ctx, domainType, domainID, group, s.authService)
}

func (s *Service) UpdateGroup(ctx context.Context, domainType, domainID, groupName string, spec *pbGroups.Group_Spec) (*pbGroups.UpdateGroupResponse, error) {
	if err := s.require(licensing.FeatureGroups); err != nil {
		return nil, err
	}

	return updateGroup(ctx, domainType, domainID, groupName, spec, s.authService)
}

func (s *Service) AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string) (*pbGroups.AddUserToGroupResponse, error) {
	if err := s.require(licensing.FeatureGroups); err != nil {
		return nil, err
	}

	return addUserToGroup(ctx, orgID, domainType, domainID, userID, userEmail, groupName, s.authService)
}

func (s *Service) require(feature licensing.Feature) error {
	if err := licensing.Require(s.entitlements, feature); err != nil {
		return grpcerrors.PermissionDenied(err, err.Error())
	}

	return nil
}
