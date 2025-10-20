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

func DescribeRole(ctx context.Context, domainType, domainID, roleName string, authService authorization.Authorization) (*pb.DescribeRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role specified")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	var roleDefinition *authorization.RoleDefinition
	var err error
	if domainID == "*" && domainType == models.DomainTypeCanvas {
		roleDefinition, err = authService.GetGlobalCanvasRoleDefinition(roleName, orgID)
	} else {
		roleDefinition, err = authService.GetRoleDefinition(roleName, domainType, domainID)
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, "role not found")
	}

	roleNames := []string{roleDefinition.Name}
	if roleDefinition.InheritsFrom != nil {
		roleNames = append(roleNames, roleDefinition.InheritsFrom.Name)
	}

	var roleMetadataMap map[string]*models.RoleMetadata
	if domainID == "*" && domainType == models.DomainTypeCanvas {
		roleMetadataMap, err = models.FindRoleMetadataByNamesWithOrgContext(roleNames, domainType, domainID, orgID)
	} else {
		roleMetadataMap, err = models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "role metadata not found")
	}

	role, err := convertRoleDefinitionToProto(roleDefinition, domainID, roleMetadataMap)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to convert role definition")
	}

	return &pb.DescribeRoleResponse{
		Role: role,
	}, nil
}
