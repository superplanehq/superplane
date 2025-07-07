package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, req *GroupUserRequest, authService authorization.Authorization) error {
	err := actions.ValidateUUIDs(req.DomainId, req.UserId)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	err = ValidateGroupUserRequest(req)
	if err != nil {
		return err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return err
	}

	err = authService.AddUserToGroup(req.DomainId, domainType, req.UserId, req.GroupName)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}