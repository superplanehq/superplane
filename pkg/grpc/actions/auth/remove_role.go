package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest, authService authorization.Authorization) (*pb.RemoveRoleResponse, error) {
	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	domainType, err := actions.ProtoToDomainType(req.RoleAssignment.DomainType)
	if err != nil {
		return nil, err
	}

	roleStr := req.RoleAssignment.Role
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	err = actions.ValidateUUIDs(req.RoleAssignment.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	userId, err := ResolveUserIDWithoutCreation(req.UserId, req.UserEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID or Email")
	}

	err = authService.RemoveRole(userId, roleStr, req.RoleAssignment.DomainId, domainType)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove role")
	}

	return &pb.RemoveRoleResponse{}, nil
}
