package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	pbUsers "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListGroupUsers(ctx context.Context, domainType, domainID, groupName string, authService authorization.Authorization) (*pb.ListGroupUsersResponse, error) {
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
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	roleMetadata := roleMetadataMap[role]

	var users []*pbUsers.User
	for _, userID := range userIDs {
		roleAssignment := &pbUsers.UserRoleAssignment{
			RoleName:        role,
			RoleDisplayName: models.GetRoleDisplayNameWithFallback(role, domainType, domainID, roleMetadata),
			RoleDescription: models.GetRoleDescriptionWithFallback(role, domainType, domainID, roleMetadata),
			DomainType:      actions.DomainTypeToProto(domainType),
			DomainId:        domainID,
			AssignedAt:      timestamppb.Now(),
		}

		user, err := convertUserToProto(userID, []*pbUsers.UserRoleAssignment{roleAssignment})
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	groupMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
	var displayName, description string
	var createdAt, updatedAt *timestamppb.Timestamp
	if err == nil {
		displayName = groupMetadata.DisplayName
		description = groupMetadata.Description
		createdAt = timestamppb.New(groupMetadata.CreatedAt)
		updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
	} else {
		displayName = groupName
		description = ""
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		},
		Spec: &pb.Group_Spec{
			Description: description,
			DisplayName: displayName,
			Role:        role,
		},
		Status: &pb.Group_Status{
			MembersCount: int32(len(userIDs)),
		},
	}

	return &pb.ListGroupUsersResponse{
		Users: users,
		Group: group,
	}, nil
}
