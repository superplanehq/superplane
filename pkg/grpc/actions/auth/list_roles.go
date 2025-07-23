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

func ListRoles(ctx context.Context, req *pb.ListRolesRequest, authService authorization.Authorization) (*pb.ListRolesResponse, error) {
	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	if req.DomainId == "" {
		return nil, status.Error(codes.InvalidArgument, "domain ID must be specified")
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	roleDefinitions, err := authService.GetAllRoleDefinitions(domainType, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve role definitions")
	}

	// Extract all role names for batch metadata lookup
	roleNames := make([]string, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		roleNames[i] = roleDef.Name
		// Also add inherited role names
		if roleDef.InheritsFrom != nil {
			roleNames = append(roleNames, roleDef.InheritsFrom.Name)
		}
	}

	// Batch fetch role metadata
	roleMetadataMap, err := models.FindRoleMetadataByNames(roleNames, domainType, req.DomainId)
	if err != nil {
		// Log error but continue with fallback behavior
		roleMetadataMap = make(map[string]*models.RoleMetadata)
	}

	roles := make([]*pb.Role, len(roleDefinitions))
	for i, roleDef := range roleDefinitions {
		role, err := convertRoleDefinitionToProto(roleDef, authService, req.DomainId, roleMetadataMap)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to convert role definition")
		}
		roles[i] = role
	}

	return &pb.ListRolesResponse{
		Roles: roles,
	}, nil
}
