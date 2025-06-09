package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CheckPermission(ctx context.Context, req *pb.CheckPermissionRequest, authService authorization.AuthorizationServiceInterface) (*pb.CheckPermissionResponse, error) {
	err := ValidateUUIDs(req.UserId, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}
	if req.Resource == "" || req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "resource and action must be specified")
	}

	var allowed bool
	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		allowed, err = authService.CheckOrganizationPermission(req.UserId, req.DomainId, req.Resource, req.Action)
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		allowed, err = authService.CheckCanvasPermission(req.UserId, req.DomainId, req.Resource, req.Action)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check permission")
	}

	return &pb.CheckPermissionResponse{
		Allowed: allowed,
	}, nil
}
