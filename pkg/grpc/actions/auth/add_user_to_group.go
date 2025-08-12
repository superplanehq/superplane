package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, orgID, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user not found")
	}

	err = authService.AddUserToGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		return nil, status.Error(codes.Unknown, err.Error())
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
