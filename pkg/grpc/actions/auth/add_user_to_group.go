package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, req *GroupUserRequest, authService authorization.Authorization) error {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	err = ValidateGroupUserRequest(req)
	if err != nil {
		return err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return err
	}

	// Handle user identification - either by user_id or user_email
	var userID string
	if req.UserID != "" {
		err := actions.ValidateUUIDs(req.UserID)
		if err != nil {
			return status.Error(codes.InvalidArgument, "invalid user ID")
		}
		userID = req.UserID
	} else if req.UserEmail != "" {
		// Find or create user by email - first try with account providers
		user, err := models.FindUserByEmail(req.UserEmail)
		if err != nil {
			// If not found by account provider, try to find inactive user by email
			user, err = models.FindInactiveUserByEmail(req.UserEmail)
			if err != nil {
				// Create inactive user with email
				user = &models.User{
					Name:     req.UserEmail,
					IsActive: false,
				}
				if err := user.Create(); err != nil {
					return status.Error(codes.Internal, "failed to create user")
				}
			}
		}
		userID = user.ID.String()
	} else {
		return status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	err = authService.AddUserToGroup(req.DomainID, domainType, userID, req.GroupName)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}
