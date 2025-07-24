package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, domainType string, domainID string, req *pbGroups.RemoveUserFromGroupRequest, authService authorization.Authorization) (*pbGroups.RemoveUserFromGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	err = ValidateGroupUserRequest(&GroupUserRequest{
		DomainID:   req.DomainId,
		DomainType: req.DomainType,
		GroupName:  req.GroupName,
		UserID:     req.UserId,
		UserEmail:  req.UserEmail,
	})
	if err != nil {
		return nil, err
	}


	var userID string
	if req.UserId != "" {
		err := actions.ValidateUUIDs(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user ID")
		}
		userID = req.UserId
	} else if req.UserEmail != "" {
		// Find user by email - first try with account providers
		user, err := models.FindUserByEmail(req.UserEmail)
		if err != nil {
			// If not found by account provider, try to find inactive user by email
			user, err = models.FindInactiveUserByEmail(req.UserEmail)
			if err != nil {
				return nil, status.Error(codes.NotFound, "user not found")
			}
		}
		userID = user.ID.String()
	} else {
		return nil, status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	err = authService.RemoveUserFromGroup(req.DomainId, domainType, userID, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pbGroups.RemoveUserFromGroupResponse{}, nil
}
