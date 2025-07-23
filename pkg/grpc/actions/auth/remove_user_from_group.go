package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, req *GroupUserRequest, authService authorization.Authorization) error {
	err := actions.ValidateUUIDs(req.DomainID, req.UserID)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	err = ValidateGroupUserRequest(req)
	if err != nil {
		return err
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return err
	}

	err = authService.RemoveUserFromGroup(req.DomainID, domainType, req.UserID, req.GroupName)
	if err != nil {
		return status.Error(codes.Internal, "failed to remove user from group")
	}

	return nil
}
