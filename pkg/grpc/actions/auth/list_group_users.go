package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	grouppb "github.com/superplanehq/superplane/pkg/protos/groups"
	userpb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroupUsers(ctx context.Context, orgID string, domainType, domainID, groupName string, authService authorization.Authorization) (*grouppb.ListGroupUsersResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	userIDs, err := authService.GetGroupUsers(domainID, domainType, groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	role, err := authService.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group roles")
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames([]string{role}, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role metadata not found")
	}

	roleMetadata := roleMetadataMap[role]

	var users []*userpb.User
	for _, userID := range userIDs {
		roleAssignment := &userpb.UserRoleAssignment{
			RoleName:        role,
			RoleDisplayName: roleMetadata.DisplayName,
			RoleDescription: roleMetadata.Description,
			DomainType:      actions.DomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      timestamppb.Now(),
		}

		user, err := convertUserToProto(orgID, userID, []*userpb.UserRoleAssignment{roleAssignment})
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	groupMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)

	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	displayName := groupMetadata.DisplayName
	description := groupMetadata.Description
	createdAt := timestamppb.New(groupMetadata.CreatedAt)
	updatedAt := timestamppb.New(groupMetadata.UpdatedAt)

	group := &grouppb.Group{
		Metadata: &grouppb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		},
		Spec: &grouppb.Group_Spec{
			Description: description,
			DisplayName: displayName,
			Role:        role,
		},
		Status: &grouppb.Group_Status{
			MembersCount: int32(len(userIDs)),
		},
	}

	return &grouppb.ListGroupUsersResponse{
		Users: users,
		Group: group,
	}, nil
}

// TODO: very inefficient way of querying the users,
// we should do a single query to get all the users
func convertUserToProto(orgID, userID string, roleAssignments []*userpb.UserRoleAssignment) (*userpb.User, error) {
	dbUser, err := models.FindUserByID(orgID, userID)
	if err != nil {
		return &userpb.User{
			Metadata: &userpb.User_Metadata{
				Id:        userID,
				Email:     "test@example.com",
				CreatedAt: timestamppb.Now(),
				UpdatedAt: timestamppb.Now(),
			},
			Spec: &userpb.User_Spec{
				DisplayName:      "Test User",
				AvatarUrl:        "",
				AccountProviders: []*userpb.AccountProvider{},
			},
			Status: &userpb.User_Status{
				IsActive:        false,
				RoleAssignments: roleAssignments,
			},
		}, nil
	}

	accountProviders, err := dbUser.GetAccountProviders()
	if err != nil {
		accountProviders = []models.AccountProvider{}
	}

	pbAccountProviders := make([]*userpb.AccountProvider, len(accountProviders))
	for i, provider := range accountProviders {
		pbAccountProviders[i] = &userpb.AccountProvider{
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

	return &userpb.User{
		Metadata: &userpb.User_Metadata{
			Id:        userID,
			Email:     dbUser.Email,
			CreatedAt: timestamppb.New(dbUser.CreatedAt),
			UpdatedAt: timestamppb.New(dbUser.UpdatedAt),
		},
		Spec: &userpb.User_Spec{
			DisplayName:      dbUser.Name,
			AvatarUrl:        getAvatarURL(accountProviders),
			AccountProviders: pbAccountProviders,
		},
		Status: &userpb.User_Status{
			RoleAssignments: roleAssignments,
		},
	}, nil
}

func getAvatarURL(accountProviders []models.AccountProvider) string {
	if len(accountProviders) == 0 {
		return ""
	}

	return accountProviders[0].AvatarURL
}
