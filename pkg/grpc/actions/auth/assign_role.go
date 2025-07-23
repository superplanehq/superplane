package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func AssignRole(ctx context.Context, req *pb.AssignRoleRequest, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var domainTypeStr string

	switch req.RoleAssignment.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		domainTypeStr = models.DomainTypeOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		domainTypeStr = models.DomainTypeCanvas
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	roleStr := req.RoleAssignment.Role
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	err := actions.ValidateUUIDs(req.RoleAssignment.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	resolvedUserID, err := ResolveUserID(req.UserId, req.UserEmail)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID or Email")
	}

	err = authService.AssignRole(resolvedUserID, roleStr, req.RoleAssignment.DomainId, domainTypeStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
