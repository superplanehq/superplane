package grpc

import (
	"context"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
)

// EnterpriseAccessControl creates and changes custom roles and groups. The
// implementation is in ee/rbac under the SuperPlane Enterprise license.
// Operations that only remove access, such as deleting a role or a group,
// stay in the Community code so that administrators can always revoke access.
type EnterpriseAccessControl interface {
	CreateRole(ctx context.Context, domainType, domainID string, role *pbRoles.Role) (*pbRoles.CreateRoleResponse, error)
	UpdateRole(ctx context.Context, domainType, domainID, roleName string, spec *pbRoles.Role_Spec) (*pbRoles.UpdateRoleResponse, error)
	CreateGroup(ctx context.Context, domainType, domainID string, group *pbGroups.Group) (*pbGroups.CreateGroupResponse, error)
	UpdateGroup(ctx context.Context, domainType, domainID, groupName string, spec *pbGroups.Group_Spec) (*pbGroups.UpdateGroupResponse, error)
	AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string) (*pbGroups.AddUserToGroupResponse, error)
}

// communityAccessControl is used when no Enterprise implementation is
// configured. It refuses every operation.
type communityAccessControl struct{}

func notLicensed() error {
	return grpcerrors.PermissionDenied(licensing.ErrNotLicensed, licensing.ErrNotLicensed.Error())
}

func (communityAccessControl) CreateRole(context.Context, string, string, *pbRoles.Role) (*pbRoles.CreateRoleResponse, error) {
	return nil, notLicensed()
}

func (communityAccessControl) UpdateRole(context.Context, string, string, string, *pbRoles.Role_Spec) (*pbRoles.UpdateRoleResponse, error) {
	return nil, notLicensed()
}

func (communityAccessControl) CreateGroup(context.Context, string, string, *pbGroups.Group) (*pbGroups.CreateGroupResponse, error) {
	return nil, notLicensed()
}

func (communityAccessControl) UpdateGroup(context.Context, string, string, string, *pbGroups.Group_Spec) (*pbGroups.UpdateGroupResponse, error) {
	return nil, notLicensed()
}

func (communityAccessControl) AddUserToGroup(context.Context, string, string, string, string, string, string) (*pbGroups.AddUserToGroupResponse, error) {
	return nil, notLicensed()
}
