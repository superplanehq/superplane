package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
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

func GetUsersWithRolesInDomain(domainID, domainType string, authService authorization.Authorization) ([]*pbUsers.User, error) {
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
		var userIDs []string

		if domainType == models.DomainTypeOrganization {
			userIDs, err = authService.GetOrgUsersForRole(roleDef.Name, domainID)
		} else {
			userIDs, err = authService.GetCanvasUsersForRole(roleDef.Name, domainID)
		}

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

	var users []*pbUsers.User
	for userID, roleAssignments := range userRoleMap {
		user, err := convertUserToProto(userID, roleAssignments)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func convertUserToProto(userID string, roleAssignments []*pbUsers.UserRoleAssignment) (*pbUsers.User, error) {
	dbUser, err := models.FindUserByID(userID)
	if err != nil {
		return &pb.User{
			Metadata: &pb.User_Metadata{
				Id:        userID,
				Email:     "test@example.com",
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
			},
			Spec: &pb.User_Spec{
				DisplayName:      "Test User",
				AvatarUrl:        "",
				AccountProviders: []*pbUsers.AccountProvider{},
			},
			Status: &pb.User_Status{
				IsActive:        false,
				RoleAssignments: roleAssignments,
			},
		}, nil
	}

	accountProviders, err := dbUser.GetAccountProviders()
	if err != nil {
		accountProviders = []models.AccountProvider{}
	}

	pbAccountProviders := make([]*pbUsers.AccountProvider, len(accountProviders))
	for i, provider := range accountProviders {
		pbAccountProviders[i] = &pb.AccountProvider{
			ProviderType: provider.Provider,
			ProviderId:   provider.ProviderID,
			Email:        provider.Email,
			DisplayName:  provider.Name,
			AvatarUrl:    provider.AvatarURL,
			IsPrimary:    i == 0, // TODO: Change when we have another login besides github
			CreatedAt:    timestamppb.New(provider.CreatedAt),
			UpdatedAt:    timestamppb.New(provider.UpdatedAt),
		}
	}

	// Determine primary email and avatar
	primaryEmail := ""
	primaryAvatar := ""
	primaryDisplayName := dbUser.Name

	if !dbUser.IsActive {
		primaryEmail = dbUser.Name
	} else if len(accountProviders) > 0 {
		primaryEmail = accountProviders[0].Email
		primaryAvatar = accountProviders[0].AvatarURL
		if primaryDisplayName == "" {
			primaryDisplayName = accountProviders[0].Name
		}
	}

	return &pb.User{
		Metadata: &pb.User_Metadata{
			Id:        userID,
			Email:     primaryEmail,
			CreatedAt: timestamppb.New(dbUser.CreatedAt),
			UpdatedAt: timestamppb.New(dbUser.UpdatedAt),
		},
		Spec: &pb.User_Spec{
			DisplayName:      primaryDisplayName,
			AvatarUrl:        primaryAvatar,
			AccountProviders: pbAccountProviders,
		},
		Status: &pb.User_Status{
			IsActive:        dbUser.IsActive,
			RoleAssignments: roleAssignments,
		},
	}, nil
}
