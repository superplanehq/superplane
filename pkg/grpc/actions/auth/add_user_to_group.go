package auth

import (
	"context"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpcerrors"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
)

func AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "user not found")
	}

	err = authService.AddUserToGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		return nil, grpcerrors.Unknown(err, "failed to add user to group")
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
