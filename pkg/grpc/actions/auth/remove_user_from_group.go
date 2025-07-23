package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, req *GroupUserRequest, authService authorization.Authorization) error {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	err = ValidateGroupUserRequest(req)
	if err != nil {
		return err
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return err
	}

	var userID string
	if req.UserID != "" {
		err := actions.ValidateUUIDs(req.UserID)
		if err != nil {
			return status.Error(codes.InvalidArgument, "invalid user ID")
		}
		userID = req.UserID
	} else if req.UserEmail != "" {
		// Find user by email - first try with account providers
		user, err := models.FindUserByEmail(req.UserEmail)
		if err != nil {
			// If not found by account provider, try to find inactive user by email
			user, err = models.FindInactiveUserByEmail(req.UserEmail)
			if err != nil {
				return status.Error(codes.NotFound, "user not found")
			}
		}
		userID = user.ID.String()
	} else {
		return status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	err = authService.RemoveUserFromGroup(req.DomainID, domainType, userID, req.GroupName)
	if err != nil {
		return status.Error(codes.Internal, "failed to remove user from group")
	}

	return nil
}
