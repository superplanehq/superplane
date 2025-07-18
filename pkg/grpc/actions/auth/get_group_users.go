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

func GetGroupUsers(ctx context.Context, req *GetGroupUsersRequest, authService authorization.Authorization) (*GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainID:   req.DomainID,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	userIDs, err := authService.GetGroupUsers(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	role, err := authService.GetGroupRole(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group roles")
	}

	// Batch fetch role metadata for the single role
	roleMetadataMap, err := models.FindRoleMetadataByNames([]string{role}, domainType, req.DomainID)
	if err != nil {
		// Log error but continue with fallback behavior
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	roleMetadata := roleMetadataMap[role]

	// Convert user IDs to User objects with role assignments
	var users []*pb.User
	for _, userID := range userIDs {
		roleAssignment := &pb.UserRoleAssignment{
			RoleName:        role,
			RoleDisplayName: models.GetRoleDisplayNameWithFallback(role, domainType, req.DomainID, roleMetadata),
			RoleDescription: models.GetRoleDescriptionWithFallback(role, domainType, req.DomainID, roleMetadata),
			DomainType:      req.DomainType,
			DomainId:        req.DomainID,
			AssignedAt:      time.Now().Format(time.RFC3339),
		}

		user, err := convertUserToProto(userID, []*pb.UserRoleAssignment{roleAssignment})
		if err != nil {
			continue // Skip users that can't be converted
		}
		users = append(users, user)
	}

	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group metadata")
	}

	group := &pb.Group{
		Name:         req.GroupName,
		DomainType:   req.DomainType,
		DomainId:     req.DomainID,
		Role:         role,
		DisplayName:  groupMetadata.DisplayName,
		Description:  groupMetadata.Description,
		MembersCount: int32(len(userIDs)),
		CreatedAt:    groupMetadata.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    groupMetadata.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return &GetGroupUsersResponse{
		Users: users,
		Group: group,
	}, nil
}
