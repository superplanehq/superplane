package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, authService authorization.Authorization, domainID string) (*pbAuth.Role, error) {
	permissions := convertPermissionsToProto(roleDef.Permissions)

	role := &pbAuth.Role{
		Name:        roleDef.Name,
		DomainType:  actions.DomainTypeToProto(roleDef.DomainType),
		Permissions: permissions,
	}

	if roleDef.InheritsFrom != nil {
		role.InheritedRole = &pbAuth.Role{
			Name:        roleDef.InheritsFrom.Name,
			DomainType:  actions.DomainTypeToProto(roleDef.InheritsFrom.DomainType),
			Permissions: convertPermissionsToProto(roleDef.InheritsFrom.Permissions),
		}
	}

	return role, nil
}

func convertPermissionsToProto(permissions []*authorization.Permission) []*pbAuth.Permission {
	permList := make([]*pbAuth.Permission, len(permissions))
	for i, perm := range permissions {
		permList[i] = convertPermissionToProto(perm)
	}
	return permList
}

func convertPermissionToProto(permission *authorization.Permission) *pbAuth.Permission {
	return &pbAuth.Permission{
		Resource:   permission.Resource,
		Action:     permission.Action,
		DomainType: actions.DomainTypeToProto(permission.DomainType),
	}
}

func SetupTestAuthService(t *testing.T) authorization.Authorization {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)
	return authService
}
