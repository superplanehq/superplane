package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListUserRoles(ctx context.Context, domainType, domainID, userID string, authService authorization.Authorization) (*pb.ListUserRolesResponse, error) {
	err := actions.ValidateUUIDs(userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	var roles []*authorization.RoleDefinition
	switch domainType {
	case models.DomainTypeOrganization:
		roles, err = authService.GetUserRolesForOrg(userID, domainID)
	case models.DomainTypeCanvas:
		if domainID == "*" {
			roles, err = authService.GetAllRoleDefinitionsWithOrgContext(models.DomainTypeCanvas, "*", orgID)
		} else {
			roles, err = authService.GetUserRolesForCanvas(userID, domainID)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
		if role.InheritsFrom != nil {
			roleNames = append(roleNames, role.InheritsFrom.Name)
		}
	}

	var roleMetadataMap map[string]*models.RoleMetadata
	if domainID == "*" {
		roleMetadataMap, err = models.FindRoleMetadataByNamesWithOrgContext(roleNames, domainType, domainID, orgID)

	} else {
		roleMetadataMap, err = models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "role metadata not found")
	}

	var rolesProto []*pbRoles.Role
	for _, role := range roles {
		roleProto, err := convertRoleDefinitionToProto(role, domainID, roleMetadataMap)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert role definition")
		}
		rolesProto = append(rolesProto, roleProto)
	}

	return &pb.ListUserRolesResponse{
		UserId:     userID,
		DomainType: actions.DomainTypeToProto(domainType),
		DomainId:   domainID,
		Roles:      rolesProto,
	}, nil
}
