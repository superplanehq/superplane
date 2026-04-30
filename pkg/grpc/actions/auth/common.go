package auth

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	userpb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, domainID string, roleMetadataMap map[string]*models.RoleMetadata) (*pbRoles.Role, error) {
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
			DisplayName: roleMetadata.DisplayName,
			Description: roleMetadata.Description,
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
				DisplayName: inheritedRoleMetadata.DisplayName,
				Description: inheritedRoleMetadata.Description,
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

func FindUser(org, id, email string) (*models.User, error) {
	if id == "" && email == "" {
		return nil, fmt.Errorf("user identifier must be specified")
	}

	orgID, err := uuid.Parse(org)
	if err != nil {
		return nil, fmt.Errorf("invalid org ID: %v", err)
	}

	if id != "" {
		userID, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %v", err)
		}

		return models.FindActiveUserByID(orgID.String(), userID.String())
	}

	return models.FindActiveUserByEmail(orgID.String(), email)
}

func usersToProto(users []models.User, accountProviders []models.UserAccountProvider) []*userpb.User {
	protoUsers := make([]*userpb.User, len(users))
	for i, user := range users {
		protoUsers[i] = &userpb.User{
			Metadata: &userpb.User_Metadata{
				Id:        user.ID.String(),
				Email:     user.GetEmail(),
				CreatedAt: timestamppb.New(user.CreatedAt),
				UpdatedAt: timestamppb.New(user.UpdatedAt),
			},
			Spec: &userpb.User_Spec{
				DisplayName: user.Name,
			},
			Status: &userpb.User_Status{
				AccountProviders: accountProvidersForUser(user.ID.String(), accountProviders),
			},
		}
	}

	return protoUsers
}

func accountProvidersForUser(userID string, accountProviders []models.UserAccountProvider) []*userpb.AccountProvider {
	providers := []*userpb.AccountProvider{}
	for _, provider := range accountProviders {
		if provider.UserID == userID {
			providers = append(providers, &userpb.AccountProvider{
				ProviderType: provider.Provider,
				ProviderId:   provider.ProviderID,
				Email:        provider.Email,
				DisplayName:  provider.Name,
				AvatarUrl:    provider.AvatarURL,
				CreatedAt:    timestamppb.New(provider.CreatedAt),
				UpdatedAt:    timestamppb.New(provider.UpdatedAt),
			})
		}
	}

	return providers
}
