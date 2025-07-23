package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pbRoles "github.com/superplanehq/superplane/pkg/protos/roles"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest, authService authorization.Authorization) (*pb.GetUserRolesResponse, error) {
	err := actions.ValidateUUIDs(req.UserId, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roles []*authorization.RoleDefinition
	switch req.DomainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		roles, err = authService.GetUserRolesForOrg(req.UserId, req.DomainId)
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		roles, err = authService.GetUserRolesForCanvas(req.UserId, req.DomainId)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	// Extract all role names for batch metadata lookup
	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
		// Also add inherited role names
		if role.InheritsFrom != nil {
			roleNames = append(roleNames, role.InheritsFrom.Name)
		}
	}

	// Batch fetch role metadata
	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, req.DomainId)
	if err != nil {
		// Log error but continue with fallback behavior
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	var rolesProto []*pbRoles.Role
	for _, role := range roles {
		roleProto, err := convertRoleDefinitionToProto(role, authService, req.DomainId, roleMetadataMap)
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
