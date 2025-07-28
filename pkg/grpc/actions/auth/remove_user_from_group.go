package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.RemoveUserFromGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	resolvedUserID, err := ResolveUserIDWithoutCreation(userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID or Email")
	}

	err = authService.RemoveUserFromGroup(domainID, domainType, resolvedUserID, groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pbGroups.RemoveUserFromGroupResponse{}, nil
}
