package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
)

func RemoveUserFromGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.RemoveUserFromGroupResponse, error) {
	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "user not found")
	}

	err = authService.RemoveUserFromGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		log.Errorf("Error removing user from group: %v", err)
		return nil, grpcerrors.Internal(err, "failed to remove user from group")
	}

	return &pbGroups.RemoveUserFromGroupResponse{}, nil
}
