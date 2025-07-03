package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetGroupUsers(ctx context.Context, req *pb.GetGroupUsersRequest, authService authorization.Authorization) (*pb.GetGroupUsersResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// Support both organization and canvas groups
	if req.DomainType != pb.DomainType_DOMAIN_TYPE_ORGANIZATION && req.DomainType != pb.DomainType_DOMAIN_TYPE_CANVAS {
		return nil, status.Error(codes.Unimplemented, "only organization and canvas groups are currently supported")
	}

	userIDs, err := authService.GetGroupUsers(req.DomainId, req.GroupName)
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

	return &pb.GetGroupUsersResponse{
		UserIds: userIDs,
		Group:   group,
	}, nil
}