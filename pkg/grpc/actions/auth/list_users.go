package auth

import (
	"context"
	"log"
	"slices"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListUsers(
	ctx context.Context,
	domainType string,
	domainID string,
	includeRoleAssignments bool,
	authService authorization.Authorization,
) (*pb.ListUsersResponse, error) {
	if domainType != models.DomainTypeOrganization {
		return nil, status.Error(codes.InvalidArgument, "domain type must be organization")
	}

	//
	// Find organization users
	//
	var users []models.User
	err := database.Conn().
		Where("organization_id = ?", domainID).
		Where("type = ?", models.UserTypeHuman).
		Find(&users).
		Error

	if err != nil {
		return nil, err
	}

	accountProviders, err := getAccountProviders(users)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch account providers")
	}

	log.Printf("accountProviders: %v", accountProviders)
	log.Printf("users: %v", users)

	protoUsers := make([]*pb.User, len(users))
	for i, user := range users {
		protoUsers[i] = &pb.User{
			Metadata: &pb.User_Metadata{
				Id:        user.ID.String(),
				Email:     user.GetEmail(),
				CreatedAt: timestamppb.New(user.CreatedAt),
				UpdatedAt: timestamppb.New(user.UpdatedAt),
			},
			Spec: &pb.User_Spec{
				DisplayName: user.Name,
			},
			Status: &pb.User_Status{
				AccountProviders: accountProvidersForUser(user.ID.String(), accountProviders),
			},
		}
	}

	if !includeRoleAssignments {
		return &pb.ListUsersResponse{
			Users: protoUsers,
		}, nil
	}

	//
	// Get all role definitions
	//
	roleDefinitions, err := authService.GetAllRoleDefinitions(domainType, domainID)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	//
	// For each role, get all users for it
	//
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
			i := slices.IndexFunc(protoUsers, func(user *pb.User) bool {
				return user.Metadata.Id == userID
			})

			if i == -1 {
				continue
			}

			protoUsers[i].Status.RoleAssignments = append(protoUsers[i].Status.RoleAssignments, roleAssignment)
		}
	}

	return &pb.ListUsersResponse{
		Users: protoUsers,
	}, nil
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

func accountProvidersForUser(userID string, accountProviders []UserAccountProvider) []*pb.AccountProvider {
	providers := []*pb.AccountProvider{}
	for _, accountProvider := range accountProviders {
		if accountProvider.UserID == userID {
			providers = append(providers, &pb.AccountProvider{
				ProviderType: accountProvider.Provider,
				ProviderId:   accountProvider.ProviderID,
			})
		}
	}

	return providers
}
