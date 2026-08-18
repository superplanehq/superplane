package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
)

func AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	requesterID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "user not found")
	}

	groupRole, err := authService.GetGroupRole(ctx, domainID, domainType, groupName)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	if err := ensureCanGrantRole(ctx, authService, requesterID, domainType, domainID, groupRole); err != nil {
		return nil, err
	}

	err = authService.AddUserToGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to add user to group")
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
