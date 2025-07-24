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

func GetGroupUsers(ctx context.Context, domainType string, domainID string, req *pb.GetGroupUsersRequest, authService authorization.Authorization) (*pb.GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainID:   req.DomainId,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}


	userIDs, err := authService.GetGroupUsers(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	role, err := authService.GetGroupRole(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group roles")
	}

	// Batch fetch role metadata for the single role
	roleMetadataMap, err := models.FindRoleMetadataByNames([]string{role}, domainType, req.DomainId)
	if err != nil {
		// Log error but continue with fallback behavior
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	roleMetadata := roleMetadataMap[role]

	// Convert user IDs to User objects with role assignments
	var users []*pbUsers.User
	for _, userID := range userIDs {
		roleAssignment := &pbUsers.UserRoleAssignment{
			RoleName:        role,
			RoleDisplayName: models.GetRoleDisplayNameWithFallback(role, domainType, req.DomainId, roleMetadata),
			RoleDescription: models.GetRoleDescriptionWithFallback(role, domainType, req.DomainId, roleMetadata),
			DomainType:      req.DomainType,
			DomainId:        req.DomainId,
			AssignedAt:      timestamppb.Now(),
		}

		user, err := convertUserToProto(userID, []*pbUsers.UserRoleAssignment{roleAssignment})
		if err != nil {
			continue // Skip users that can't be converted
		}
		users = append(users, user)
	}

	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainId)
	var displayName, description string
	var createdAt, updatedAt *timestamppb.Timestamp
	if err == nil {
		displayName = groupMetadata.DisplayName
		description = groupMetadata.Description
		createdAt = timestamppb.New(groupMetadata.CreatedAt)
		updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
	} else {
		// Use fallback values when metadata is not found
		displayName = req.GroupName
		description = ""
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       req.GroupName,
			DomainType: req.DomainType,
			DomainId:   req.DomainId,
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

	return &pb.GetGroupUsersResponse{
		Users: users,
		Group: group,
	}, nil
}
