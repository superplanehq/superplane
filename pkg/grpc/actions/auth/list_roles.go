package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func ListRoles(ctx context.Context, domainType string, domainID string, authService authorization.Authorization) (*pb.ListRolesResponse, error) {
	roleDefinitions, err := authService.GetAllRoleDefinitions(ctx, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to retrieve role definitions")
	}

	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
		if roleDef.InheritsFrom != nil {
			roleNames = append(roleNames, roleDef.InheritsFrom.Name)
		}
	}

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "role not found")
	}

	roles := make([]*pb.Role, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		role, err := convertRoleDefinitionToProto(roleDef, domainID, roleMetadataMap)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to convert role definition")
		}
		roles[i] = role
	}

	return &pb.ListRolesResponse{
		Roles: roles,
	}, nil
}
