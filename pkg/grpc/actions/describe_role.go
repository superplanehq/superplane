package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest, authService authorization.AuthorizationServiceInterface) (*pb.DescribeRoleResponse, error) {
	// Validate domain type and role
	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var role *pb.Role
	var roleStr string

	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		if req.GetOrgRole() == pb.OrganizationRole_ORG_ROLE_UNSPECIFIED {
			return nil, status.Error(codes.InvalidArgument, "organization role must be specified")
		}
		roleStr = convertOrgRoleToString(req.GetOrgRole())
		role = buildOrgRole(roleStr, req.DomainType)
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		if req.GetCanvasRole() == pb.CanvasRole_CANVAS_ROLE_UNSPECIFIED {
			return nil, status.Error(codes.InvalidArgument, "canvas role must be specified")
		}
		roleStr = convertCanvasRoleToString(req.GetCanvasRole())
		role = buildCanvasRole(roleStr, req.DomainType)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if role == nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	return &pb.DescribeRoleResponse{
		Role: role,
	}, nil
}
