package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest, authService authorization.Authorization) (*pb.GetUserRolesResponse, error) {
	err := actions.ValidateUUIDs(req.UserId, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roles []*authorization.RoleDefinition
	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		roles, err = authService.GetUserRolesForOrg(req.UserId, req.DomainId)
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		roles, err = authService.GetUserRolesForCanvas(req.UserId, req.DomainId)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	var rolesProto []*pb.Role
	for _, role := range roles {
		roleProto, err := convertRoleDefinitionToProto(role, authService, req.DomainId)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert role definition")
		}
		rolesProto = append(rolesProto, roleProto)
	}

	return &pb.GetUserRolesResponse{
		UserId:     req.UserId,
		DomainType: req.DomainType,
		DomainId:   req.DomainId,
		Roles:      rolesProto,
	}, nil
}
