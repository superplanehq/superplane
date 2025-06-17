package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AssignRole(ctx context.Context, req *pb.AssignRoleRequest, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	err := actions.ValidateUUIDs(req.UserId, req.RoleAssignment.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var domainTypeStr string

	switch req.RoleAssignment.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		domainTypeStr = authorization.DomainOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		domainTypeStr = authorization.DomainCanvas
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	roleStr := req.RoleAssignment.Role
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	// TODO: once users are implemented, check if the user exists

	err = authService.AssignRole(req.UserId, roleStr, req.RoleAssignment.DomainId, domainTypeStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
