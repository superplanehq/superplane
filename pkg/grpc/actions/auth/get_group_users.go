package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetGroupUsers(ctx context.Context, req *GetGroupUsersRequest, authService authorization.Authorization) (*GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainId:   req.DomainId,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	userIDs, err := authService.GetGroupUsers(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	// Create group object for response
	group := &pb.Group{
		Name:       req.GroupName,
		DomainType: req.DomainType,
		DomainId:   req.DomainId,
		Role:       "", // TODO: get actual role from service
	}

	return &GetGroupUsersResponse{
		UserIds: userIDs,
		Group:   group,
	}, nil
}
