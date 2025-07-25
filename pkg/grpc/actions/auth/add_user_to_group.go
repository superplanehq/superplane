package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	userID, err := ResolveUserID(userID, userEmail)
	if err != nil {
		return nil, err
	}

	err = authService.AddUserToGroup(domainID, domainType, userID, groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
