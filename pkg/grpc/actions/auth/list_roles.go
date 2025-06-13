package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListRoles(ctx context.Context, req *pb.ListRolesRequest, authService authorization.Authorization) (*pb.ListRolesResponse, error) {
	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	if req.DomainId == "" {
		return nil, status.Error(codes.InvalidArgument, "domain ID must be specified")
	}

	domainType := convertDomainType(req.DomainType)
	if domainType == "" {
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	roleDefinitions, err := authService.GetAllRoleDefinitions(domainType, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve role definitions")
	}

	roles := make([]*pb.Role, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		role, err := convertRoleDefinitionToProto(roleDef, authService, req.DomainId)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert role definition")
		}
		roles[i] = role
	}

	return &pb.ListRolesResponse{
		Roles: roles,
	}, nil
}
