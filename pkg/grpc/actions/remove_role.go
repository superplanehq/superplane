package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest, authService authorization.AuthorizationServiceInterface) (*pb.RemoveRoleResponse, error) {
	err := ValidateUUIDs(req.UserId, req.RoleAssignment.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roleStr string
	var domainTypeStr string

	switch req.RoleAssignment.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		if req.RoleAssignment.GetOrgRole() == pb.OrganizationRole_ORG_ROLE_UNSPECIFIED {
			return nil, status.Error(codes.InvalidArgument, "organization role must be specified")
		}
		roleStr = convertOrgRoleToString(req.RoleAssignment.GetOrgRole())
		domainTypeStr = authorization.DomainOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		if req.RoleAssignment.GetCanvasRole() == pb.CanvasRole_CANVAS_ROLE_UNSPECIFIED {
			return nil, status.Error(codes.InvalidArgument, "canvas role must be specified")
		}
		roleStr = convertCanvasRoleToString(req.RoleAssignment.GetCanvasRole())
		domainTypeStr = authorization.DomainCanvas
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	// Remove the role
	err = authService.RemoveRole(req.UserId, roleStr, req.RoleAssignment.DomainId, domainTypeStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove role")
	}

	return &pb.RemoveRoleResponse{}, nil
}
