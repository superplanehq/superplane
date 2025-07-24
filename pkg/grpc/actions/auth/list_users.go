package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListUsers(ctx context.Context, domainType string, domainID string, req *pb.ListUsersRequest, authService authorization.Authorization) (*pb.ListUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	users, err := GetUsersWithRolesInDomain(req.DomainId, domainType, authService)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get canvas users")
	}

	return &pb.ListUsersResponse{
		Users: users,
	}, nil
}

func GetUsersWithRolesInDomain(domainID, domainType string, authService authorization.Authorization) ([]*pb.User, error) {
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
		// Log error but continue with fallback behavior
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	userRoleMap := make(map[string][]*pb.UserRoleAssignment)

	for _, roleDef := range roleDefinitions {
		var userIDs []string

		if domainType == models.DomainTypeOrg {
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
			RoleDisplayName: models.GetRoleDisplayNameWithFallback(roleDef.Name, domainType, domainID, roleMetadata),
			RoleDescription: models.GetRoleDescriptionWithFallback(roleDef.Name, domainType, domainID, roleMetadata),
			DomainType:      actions.DomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      timestamppb.Now(),
		}

		for _, userID := range userIDs {
			userRoleMap[userID] = append(userRoleMap[userID], roleAssignment)
		}
	}

	var users []*pb.User
	for userID, roleAssignments := range userRoleMap {
		user, err := convertUserToProto(userID, roleAssignments)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func convertUserToProto(userID string, roleAssignments []*pb.UserRoleAssignment) (*pb.User, error) {
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
				AccountProviders: []*pb.AccountProvider{},
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

	pbAccountProviders := make([]*pb.AccountProvider, len(accountProviders))
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
