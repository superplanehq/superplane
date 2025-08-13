package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.RemoveUserFromGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}

	err = authService.RemoveUserFromGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		log.Errorf("Error removing user from group: %v", err)
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pbGroups.RemoveUserFromGroupResponse{}, nil
}
