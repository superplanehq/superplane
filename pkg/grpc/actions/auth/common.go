package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, authService authorization.Authorization, domainID string, roleMetadataMap map[string]*models.RoleMetadata) (*pbRoles.Role, error) {
	permissions := convertPermissionsToProto(roleDef.Permissions)

	roleMetadata := roleMetadataMap[roleDef.Name]
	role := &pbRoles.Role{
		Metadata: &pbRoles.Role_Metadata{
			Name:       roleDef.Name,
			DomainType: actions.DomainTypeToProto(roleDef.DomainType),
			DomainId:   domainID,
			CreatedAt:  timestamppb.New(roleMetadataMap[roleDef.Name].CreatedAt),
			UpdatedAt:  timestamppb.New(roleMetadataMap[roleDef.Name].UpdatedAt),
		},
		Spec: &pbRoles.Role_Spec{
			DisplayName: models.GetRoleDisplayNameWithFallback(roleDef.Name, roleDef.DomainType, domainID, roleMetadata),
			Description: models.GetRoleDescriptionWithFallback(roleDef.Name, roleDef.DomainType, domainID, roleMetadata),
			Permissions: permissions,
		},
	}

	if roleDef.InheritsFrom != nil {
		inheritedRoleMetadata := roleMetadataMap[roleDef.InheritsFrom.Name]
		role.Spec.InheritedRole = &pbRoles.Role{
			Metadata: &pbRoles.Role_Metadata{
				Name:       roleDef.InheritsFrom.Name,
				DomainType: actions.DomainTypeToProto(roleDef.InheritsFrom.DomainType),
				DomainId:   domainID,
				CreatedAt:  timestamppb.New(inheritedRoleMetadata.CreatedAt),
				UpdatedAt:  timestamppb.New(inheritedRoleMetadata.UpdatedAt),
			},
			Spec: &pbRoles.Role_Spec{
				DisplayName: models.GetRoleDisplayNameWithFallback(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID, inheritedRoleMetadata),
				Description: models.GetRoleDescriptionWithFallback(roleDef.InheritsFrom.Name, roleDef.InheritsFrom.DomainType, domainID, inheritedRoleMetadata),
				Permissions: convertPermissionsToProto(roleDef.InheritsFrom.Permissions),
			},
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
