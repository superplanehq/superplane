package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pbGroups "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AddUserToGroup(ctx context.Context, domainType string, domainID string, req *pbGroups.AddUserToGroupRequest, authService authorization.Authorization) (*pbGroups.AddUserToGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	err = ValidateGroupUserRequest(&GroupUserRequest{
		DomainID:   req.DomainId,
		DomainType: req.DomainType,
		GroupName:  req.GroupName,
		UserID:     req.UserId,
		UserEmail:  req.UserEmail,
	})
	if err != nil {
		return nil, err
	}


	// Handle user identification using shared function
	userID, err := ResolveUserID(req.UserId, req.UserEmail)
	if err != nil {
		return nil, err
	}

	err = authService.AddUserToGroup(req.DomainId, domainType, userID, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pbGroups.AddUserToGroupResponse{}, nil
}
