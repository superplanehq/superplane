package auth

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func GetUsersWithRolesInDomain(domainID, domainType string, includeServiceAccounts bool, authService authorization.Authorization) ([]*pbUsers.User, error) {
	if domainType != models.DomainTypeOrganization {
		return nil, status.Error(codes.InvalidArgument, "domain type must be organization")
	}

	roleDefinitions, err := authService.GetAllRoleDefinitions(domainType, domainID)
	if err != nil {
		return nil, err
	}

	// Extract all role names for batch metadata lookup
	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
	}

	// Batch fetch role metadata
	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	userRoleMap := make(map[string][]*pbUsers.UserRoleAssignment)

	for _, roleDef := range roleDefinitions {
		userIDs, err := authService.GetOrgUsersForRole(roleDef.Name, domainID)
		if err != nil {
			continue
		}

		roleMetadata := roleMetadataMap[roleDef.Name]
		roleAssignment := &pb.UserRoleAssignment{
			RoleName:        roleDef.Name,
			RoleDisplayName: roleMetadata.DisplayName,
			RoleDescription: roleMetadata.Description,
			DomainType:      actions.DomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      timestamppb.Now(),
		}

		for _, userID := range userIDs {
			userRoleMap[userID] = append(userRoleMap[userID], roleAssignment)
		}
	}

	userIDs := make([]string, 0, len(userRoleMap))
	for userID := range userRoleMap {
		userIDs = append(userIDs, userID)
	}

	var dbUsers []models.User
	if includeServiceAccounts {
		dbUsers, err = models.FindUsersByIDs(userIDs)
	} else {
		dbUsers, err = models.FindHumanUsersByIDs(userIDs)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch users")
	}

	var users []*pbUsers.User
	for i := range dbUsers {
		roleAssignments := userRoleMap[dbUsers[i].ID.String()]
		user, err := convertUserToProto(&dbUsers[i], roleAssignments)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func convertUserToProto(dbUser *models.User, roleAssignments []*pbUsers.UserRoleAssignment) (*pbUsers.User, error) {
	var pbAccountProviders []*pbUsers.AccountProvider

	if dbUser.AccountID != nil {
		account, err := models.FindAccountByID(dbUser.AccountID.String())
		if err == nil {
			providers, err := account.GetAccountProviders()
			if err == nil {
				pbAccountProviders = make([]*pbUsers.AccountProvider, len(providers))
				for i, provider := range providers {
					pbAccountProviders[i] = &pb.AccountProvider{
						ProviderType: provider.Provider,
						ProviderId:   provider.ProviderID,
						Email:        provider.Email,
						DisplayName:  provider.Name,
						AvatarUrl:    provider.AvatarURL,
						CreatedAt:    timestamppb.New(provider.CreatedAt),
						UpdatedAt:    timestamppb.New(provider.UpdatedAt),
					}
				}
			}
		}
	}

	return &pb.User{
		Metadata: &pb.User_Metadata{
			Id:        dbUser.ID.String(),
			Email:     dbUser.GetEmail(),
			CreatedAt: timestamppb.New(dbUser.CreatedAt),
			UpdatedAt: timestamppb.New(dbUser.UpdatedAt),
		},
		Spec: &pb.User_Spec{
			DisplayName:      dbUser.Name,
			AccountProviders: pbAccountProviders,
		},
		Status: &pb.User_Status{
			RoleAssignments: roleAssignments,
		},
	}, nil
}
