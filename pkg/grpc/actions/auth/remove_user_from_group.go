package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest, authService authorization.Authorization) (*pb.RemoveUserFromGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId, req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
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

	err = authService.RemoveUserFromGroup(req.DomainId, req.UserId, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove user from group")
	}

	return &pb.RemoveUserFromGroupResponse{}, nil
}