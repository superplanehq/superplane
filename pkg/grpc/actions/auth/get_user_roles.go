package auth

import (
	"context"

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

	var roles []*authorization.RoleDefinition
	switch domainType {
	case models.DomainTypeOrg:
		roles, err = authService.GetUserRolesForOrg(userID, domainID)
	case models.DomainTypeCanvas:
		roles, err = authService.GetUserRolesForCanvas(userID, domainID)
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

	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, domainID)
	if err != nil {
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	var rolesProto []*pbRoles.Role
	for _, role := range roles {
		roleProto, err := convertRoleDefinitionToProto(role, authService, domainID, roleMetadataMap)
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
