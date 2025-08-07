package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.RemoveUserFromGroupResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)

	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID or Email")
	}

	err = authService.RemoveUserFromGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pbGroups.RemoveUserFromGroupResponse{}, nil
}
