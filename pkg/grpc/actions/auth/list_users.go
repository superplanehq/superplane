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

func ListUsers(ctx context.Context, orgID, domainType, domainID string, authService authorization.Authorization) (*pb.ListUsersResponse, error) {
	users, err := GetUsersWithRolesInDomain(orgID, domainID, domainType, authService)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get canvas users")
	}

	return &pb.ListUsersResponse{
		Users: users,
	}, nil
}

func GetUsersWithRolesInDomain(orgID string, domainID, domainType string, authService authorization.Authorization) ([]*pb.User, error) {
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

	userRoleMap := make(map[string][]*pb.UserRoleAssignment)

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

	var users []*pb.User
	for userID, roleAssignments := range userRoleMap {
		user, err := convertUserToProto(orgID, userID, roleAssignments)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	return users, nil
}
