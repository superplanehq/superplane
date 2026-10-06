package enterprise

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
)

func TestNewRegistryReturnsCommunityRbac(t *testing.T) {
	registry := NewRegistry()

	rbac, err := Get[Rbac](registry, RBAC)
	require.NoError(t, err)

	_, err = rbac.CreateRole(t.Context(), "organization", "org", nil)
	require.Error(t, err)
}

func TestGetRejectsAMissingOrWrongCapability(t *testing.T) {
	registry := NewRegistry()

	_, err := Get[Rbac](nil, RBAC)
	require.Error(t, err)

	_, err = Get[Rbac](registry, "missing")
	require.Error(t, err)

	registry.Register(RBAC, "not an implementation")
	_, err = Get[Rbac](registry, RBAC)
	require.Error(t, err)
}

func TestRegisterReplacesTheCommunityImplementation(t *testing.T) {
	registry := NewRegistry()
	registry.Register(RBAC, stubRbac{})

	rbac, err := Get[Rbac](registry, RBAC)
	require.NoError(t, err)
	_, err = rbac.CreateRole(t.Context(), "organization", "org", nil)
	require.NoError(t, err)
}

type stubRbac struct{}

func (stubRbac) CreateRole(context.Context, string, string, *pbRoles.Role) (*pbRoles.CreateRoleResponse, error) {
	return &pbRoles.CreateRoleResponse{}, nil
}

func (stubRbac) UpdateRole(context.Context, string, string, string, *pbRoles.Role_Spec) (*pbRoles.UpdateRoleResponse, error) {
	return nil, nil
}

func (stubRbac) CreateGroup(context.Context, string, string, *pbGroups.Group) (*pbGroups.CreateGroupResponse, error) {
	return nil, nil
}

func (stubRbac) UpdateGroup(context.Context, string, string, string, *pbGroups.Group_Spec) (*pbGroups.UpdateGroupResponse, error) {
	return nil, nil
}

func (stubRbac) AddUserToGroup(context.Context, string, string, string, string, string, string) (*pbGroups.AddUserToGroupResponse, error) {
	return nil, nil
}
