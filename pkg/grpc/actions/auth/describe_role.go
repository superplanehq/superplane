package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func DescribeRole(ctx context.Context, domainType, domainID, roleName string, authService authorization.Authorization) (*pb.DescribeRoleResponse, error) {
	if roleName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "invalid role specified")
	}

	roleDefinition, err := authService.GetRoleDefinition(ctx, roleName, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "role not found")
	}

	roleNames := []string{roleDefinition.Name}
	if roleDefinition.InheritsFrom != nil {
		roleNames = append(roleNames, roleDefinition.InheritsFrom.Name)
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "role metadata not found")
	}

	role, err := convertRoleDefinitionToProto(roleDefinition, domainID, roleMetadataMap)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to convert role definition")
	}

	return &pb.DescribeRoleResponse{
		Role: role,
	}, nil
}
