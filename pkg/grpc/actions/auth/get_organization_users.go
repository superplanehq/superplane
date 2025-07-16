package auth

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetOrganizationUsers(ctx context.Context, req *pb.GetOrganizationUsersRequest, authService authorization.Authorization) (*pb.GetOrganizationUsersResponse, error) {
	err := actions.ValidateUUIDs(req.OrganizationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	// Get all users with roles in the organization
	users, err := GetUsersWithRolesInDomain(req.OrganizationId, authorization.DomainOrg, authService)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get organization users")
	}

	return &pb.GetOrganizationUsersResponse{
		Users: users,
	}, nil
}

func GetUsersWithRolesInDomain(domainID, domainType string, authService authorization.Authorization) ([]*pb.User, error) {
	roleDefinitions, err := authService.GetAllRoleDefinitions(domainType, domainID)
	if err != nil {
		return nil, err
	}

	userRoleMap := make(map[string][]*pb.UserRoleAssignment)

	for _, roleDef := range roleDefinitions {
		var userIDs []string

		if domainType == authorization.DomainOrg {
			userIDs, err = authService.GetOrgUsersForRole(roleDef.Name, domainID)
		} else {
			userIDs, err = authService.GetCanvasUsersForRole(roleDef.Name, domainID)
		}

		if err != nil {
			continue
		}

		roleAssignment := &pb.UserRoleAssignment{
			RoleName:        roleDef.Name,
			RoleDisplayName: models.GetRoleDisplayName(roleDef.Name, domainType, domainID),
			RoleDescription: models.GetRoleDescription(roleDef.Name, domainType, domainID),
			DomainType:      convertDomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      time.Now().Format(time.RFC3339),
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
			UserId:           userID,
			DisplayName:      "Test User",
			Email:            "test@example.com",
			AvatarUrl:        "",
			IsActive:         true,
			CreatedAt:        time.Now().Format(time.RFC3339),
			UpdatedAt:        time.Now().Format(time.RFC3339),
			RoleAssignments:  roleAssignments,
			AccountProviders: []*pb.AccountProvider{},
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
			CreatedAt:    provider.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    provider.UpdatedAt.Format(time.RFC3339),
		}
	}

	// Determine primary email and avatar
	primaryEmail := ""
	primaryAvatar := ""
	primaryDisplayName := dbUser.Name

	if len(accountProviders) > 0 {
		primaryEmail = accountProviders[0].Email
		primaryAvatar = accountProviders[0].AvatarURL
		if primaryDisplayName == "" {
			primaryDisplayName = accountProviders[0].Name
		}
	}

	return &pb.User{
		UserId:           userID,
		DisplayName:      primaryDisplayName,
		Email:            primaryEmail,
		AvatarUrl:        primaryAvatar,
		IsActive:         true, // TODO: Add active status to user model
		CreatedAt:        dbUser.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        dbUser.UpdatedAt.Format(time.RFC3339),
		RoleAssignments:  roleAssignments,
		AccountProviders: pbAccountProviders,
	}, nil
}
