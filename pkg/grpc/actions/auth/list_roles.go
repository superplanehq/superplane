package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListRoles(ctx context.Context, domainType string, domainID string, authService authorization.Authorization) (*pb.ListRolesResponse, error) {
	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	var roleDefinitions []*authorization.RoleDefinition
	var err error
	if domainID == "*" {
		roleDefinitions, err = authService.GetAllRoleDefinitionsWithOrgContext(domainType, "*", orgID)
	} else {
		roleDefinitions, err = authService.GetAllRoleDefinitions(domainType, domainID)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve role definitions")
	}

	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
		if roleDef.InheritsFrom != nil {
			roleNames = append(roleNames, roleDef.InheritsFrom.Name)
		}
	}

	var roleMetadataMap map[string]*models.RoleMetadata
	if domainID == "*" {
		roleMetadataMap, err = models.FindRoleMetadataByNamesWithOrgContext(roleNames, domainType, domainID, orgID)
	} else {
		roleMetadataMap, err = models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	roles := make([]*pb.Role, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		role, err := convertRoleDefinitionToProto(roleDef, domainID, roleMetadataMap)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert role definition")
		}
		roles[i] = role
	}

	return &pb.ListRolesResponse{
		Roles: roles,
	}, nil
}
