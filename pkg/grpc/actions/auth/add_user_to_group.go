package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
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

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return err
	}

	// Handle user identification using shared function
	userID, err := ResolveUserID(req.UserID, req.UserEmail)
	if err != nil {
		return err
	}

	err = authService.AddUserToGroup(req.DomainID, domainType, userID, req.GroupName)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}
