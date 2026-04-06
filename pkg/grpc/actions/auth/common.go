package auth

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
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

func usersToProto(users []models.User, accountProviders []UserAccountProvider) []*userpb.User {
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

type UserAccountProvider struct {
	models.AccountProvider
	UserID string
}

func getAccountProviders(users []models.User) ([]UserAccountProvider, error) {
	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.ID.String()
	}

	var accountProviders []UserAccountProvider
	err := database.Conn().
		Table("users").
		Select("users.id as user_id, account_providers.*").
		Joins("inner join accounts on accounts.id = users.account_id").
		Joins("inner join account_providers on account_providers.account_id = accounts.id").
		Where("users.id IN (?)", userIDs).
		Find(&accountProviders).
		Error

	if err != nil {
		return nil, err
	}

	return accountProviders, nil
}

func accountProvidersForUser(userID string, accountProviders []UserAccountProvider) []*userpb.AccountProvider {
	providers := []*userpb.AccountProvider{}
	for _, accountProvider := range accountProviders {
		if accountProvider.UserID == userID {
			providers = append(providers, &userpb.AccountProvider{
				ProviderType: accountProvider.Provider,
				ProviderId:   accountProvider.ProviderID,
			})
		}
	}

	return providers
}
