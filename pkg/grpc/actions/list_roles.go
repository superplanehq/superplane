package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListRoles(ctx context.Context, req *pb.ListRolesRequest, authService authorization.AuthorizationServiceInterface) (*pb.ListRolesResponse, error) {
	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roles []*pb.Role

	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		roles = []*pb.Role{
			buildOrgRole(authorization.RoleOrgViewer, req.DomainType),
			buildOrgRole(authorization.RoleOrgAdmin, req.DomainType),
			buildOrgRole(authorization.RoleOrgOwner, req.DomainType),
		}
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		roles = []*pb.Role{
			buildCanvasRole(authorization.RoleCanvasViewer, req.DomainType),
			buildCanvasRole(authorization.RoleCanvasAdmin, req.DomainType),
			buildCanvasRole(authorization.RoleCanvasOwner, req.DomainType),
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	return &pb.ListRolesResponse{
		Roles: roles,
	}, nil
}
