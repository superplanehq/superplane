package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeRole(ctx context.Context, req *pb.DescribeRoleRequest, authService authorization.Authorization) (*pb.DescribeRoleResponse, error) {
	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	if req.DomainId == "" {
		return nil, status.Error(codes.InvalidArgument, "domain ID must be specified")
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role specified")
	}

	roleDefinition, err := authService.GetRoleDefinition(req.Role, domainType, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	roleNames := []string{roleDefinition.Name}
	if roleDefinition.InheritsFrom != nil {
		roleNames = append(roleNames, roleDefinition.InheritsFrom.Name)
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, req.DomainId)
	if err != nil {
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	role, err := convertRoleDefinitionToProto(roleDefinition, authService, req.DomainId, roleMetadataMap)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert role definition")
	}

	return &pb.DescribeRoleResponse{
		Role: role,
	}, nil
}
