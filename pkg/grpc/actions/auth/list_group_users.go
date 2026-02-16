package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
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
		log.Errorf("failed to get group users: %v", err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	role, err := authService.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group role: %v", err)
		return nil, status.Error(codes.Internal, "failed to get group role")
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames([]string{role}, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role metadata not found")
	}

	roleMetadata := roleMetadataMap[role]

	dbUsers, err := models.FindUsersByIDs(userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch group users")
	}

	roleAssignment := &pbUsers.UserRoleAssignment{
		RoleName:        role,
		RoleDisplayName: roleMetadata.DisplayName,
		RoleDescription: roleMetadata.Description,
		DomainType:      actions.DomainTypeToProto(domainType),
		DomainId:        domainID,
		AssignedAt:      timestamppb.Now(),
	}

	var users []*pbUsers.User
	for i := range dbUsers {
		user, err := convertUserToProto(&dbUsers[i], []*pbUsers.UserRoleAssignment{roleAssignment})
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
