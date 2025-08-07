package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, domainType, domainID, userID, userEmail, groupName string, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	orgID := ctx.Value(authorization.OrganizationContextKey).(string)

	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	user, err := FindUser(orgID, userID, userEmail)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not found: %v", err)
	}

	err = authService.AddUserToGroup(domainID, domainType, user.ID.String(), groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
