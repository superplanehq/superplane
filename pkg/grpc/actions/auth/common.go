package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func convertDomainType(domainType pbAuth.DomainType) string {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return models.DomainOrg
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		return models.DomainCanvas
	default:
		return ""
	}
}

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, authService authorization.Authorization, domainID string, roleMetadataMap map[string]*models.RoleMetadata) (*pbAuth.Role, error) {
	permissions := convertPermissionsToProto(roleDef.Permissions)

	roleMetadata := roleMetadataMap[roleDef.Name]
	role := &pbAuth.Role{
		Name:        roleDef.Name,
		DomainType:  convertDomainTypeToProto(roleDef.DomainType),
		Permissions: permissions,
		DisplayName: models.GetRoleDisplayNameWithFallback(roleDef.Name, roleDef.DomainType, domainID, roleMetadata),
		Description: models.GetRoleDescriptionWithFallback(roleDef.Name, roleDef.DomainType, domainID, roleMetadata),
	}

	if roleDef.InheritsFrom != nil {
		inheritedRoleMetadata := roleMetadataMap[roleDef.InheritsFrom.Name]
		role.InheritedRole = &pbAuth.Role{
			Name:        roleDef.InheritsFrom.Name,
			DomainType:  convertDomainTypeToProto(roleDef.InheritsFrom.DomainType),
			Permissions: convertPermissionsToProto(roleDef.InheritsFrom.Permissions),
			DisplayName: models.GetRoleDisplayNameWithFallback(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID, inheritedRoleMetadata),
			Description: models.GetRoleDescriptionWithFallback(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID, inheritedRoleMetadata),
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
	case models.DomainOrg:
		return pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION
	case models.DomainCanvas:
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

func ResolveUserID(userID, userEmail string) (string, error) {
	if userID == "" && userEmail == "" {
		return "", status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	if userID != "" {
		return resolveByID(userID)
	}

	return resolveByEmail(userEmail, true)
}

func ResolveUserIDWithoutCreation(userID, userEmail string) (string, error) {
	if userID == "" && userEmail == "" {
		return "", status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	if userID != "" {
		return resolveByID(userID)
	}

	return resolveByEmail(userEmail, false)
}

func resolveByID(userID string) (string, error) {
	if err := actions.ValidateUUIDs(userID); err != nil {
		return "", status.Error(codes.InvalidArgument, "invalid user ID")
	}

	if _, err := models.FindUserByID(userID); err != nil {
		return "", status.Error(codes.NotFound, "user not found")
	}

	return userID, nil
}

func resolveByEmail(userEmail string, create bool) (string, error) {
	if !create {
		user, err := models.FindUserByEmail(userEmail)
		if err != nil {
			return "", status.Error(codes.NotFound, "user not found")
		}
		return user.ID.String(), nil
	}

	user, err := findOrCreateUser(userEmail)
	if err != nil {
		return "", err
	}
	return user.ID.String(), nil
}

func findOrCreateUser(email string) (*models.User, error) {
	if user, err := models.FindUserByEmail(email); err == nil {
		return user, nil
	}

	if user, err := models.FindInactiveUserByEmail(email); err == nil {
		return user, nil
	}

	user := &models.User{
		Name:     email,
		IsActive: false,
	}

	if err := user.Create(); err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return user, nil
}
