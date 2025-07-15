package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
)

func convertDomainType(domainType pbAuth.DomainType) string {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return authorization.DomainOrg
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		return authorization.DomainCanvas
	default:
		return ""
	}
}

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, authService authorization.Authorization, domainID string) (*pbAuth.Role, error) {
	permissions := convertPermissionsToProto(roleDef.Permissions)

	role := &pbAuth.Role{
		Name:        roleDef.Name,
		DomainType:  convertDomainTypeToProto(roleDef.DomainType),
		Permissions: permissions,
		DisplayName: models.GetRoleDisplayName(roleDef.Name, roleDef.DomainType, domainID),
		Description: models.GetRoleDescription(roleDef.Name, roleDef.DomainType, domainID),
	}

	if roleDef.InheritsFrom != nil {
		role.InheritedRole = &pbAuth.Role{
			Name:        roleDef.InheritsFrom.Name,
			DomainType:  convertDomainTypeToProto(roleDef.InheritsFrom.DomainType),
			Permissions: convertPermissionsToProto(roleDef.InheritsFrom.Permissions),
			DisplayName: models.GetRoleDisplayName(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID),
			Description: models.GetRoleDescription(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID),
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
		DomainType: convertDomainTypeToProto(permission.DomainType),
	}
}

func convertDomainTypeToProto(domainType string) pbAuth.DomainType {
	switch domainType {
	case authorization.DomainOrg:
		return pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION
	case authorization.DomainCanvas:
		return pbAuth.DomainType_DOMAIN_TYPE_CANVAS
	default:
		return pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED
	}
}

func SetupTestAuthService(t *testing.T) authorization.Authorization {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)
	return authService
}

func CreateGroupWithMetadata(domainID, domainType, groupName, role, displayName, description string, authService authorization.Authorization) error {
	err := authService.CreateGroup(domainID, domainType, groupName, role)
	if err != nil {
		return err
	}

	return models.UpsertGroupMetadata(groupName, domainType, domainID, displayName, description)
}

func CreateRoleWithMetadata(domainID string, roleDef *authorization.RoleDefinition, displayName, description string, authService authorization.Authorization) error {
	err := authService.CreateCustomRole(domainID, roleDef)
	if err != nil {
		return err
	}

	return models.UpsertRoleMetadata(roleDef.Name, roleDef.DomainType, domainID, displayName, description)
}
