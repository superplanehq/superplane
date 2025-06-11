package actions

import (
	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		_, err := uuid.Parse(id)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid UUID: %s", id)
		}
	}

	return nil
}

func convertDomainType(domainType pb.DomainType) string {
	switch domainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return authorization.DomainOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		return authorization.DomainCanvas
	default:
		return ""
	}
}

func convertRoleDefinitionToProto(roleDef *authorization.RoleDefinition, authService authorization.AuthorizationServiceInterface, domainID string) (*pb.Role, error) {
	permissions := convertPermissionsToProto(roleDef.Permissions)

	role := &pb.Role{
		Name:        roleDef.Name,
		DomainType:  convertDomainTypeToProto(roleDef.DomainType),
		Permissions: permissions,
	}

	if roleDef.InheritsFrom != nil {
		role.InheritedRole = &pb.Role{
			Name:        roleDef.InheritsFrom.Name,
			DomainType:  convertDomainTypeToProto(roleDef.InheritsFrom.DomainType),
			Permissions: convertPermissionsToProto(roleDef.InheritsFrom.Permissions),
		}
	}

	return role, nil
}

func convertPermissionsToProto(permissions []*authorization.Permission) []*pb.Permission {
	permList := make([]*pb.Permission, len(permissions))
	for i, perm := range permissions {
		permList[i] = convertPermissionToProto(perm)
	}
	return permList
}

func convertPermissionToProto(permission *authorization.Permission) *pb.Permission {
	return &pb.Permission{
		Resource:   permission.Resource,
		Action:     permission.Action,
		DomainType: convertDomainTypeToProto(permission.DomainType),
	}
}

func convertDomainTypeToProto(domainType string) pb.DomainType {
	switch domainType {
	case authorization.DomainOrg:
		return pb.DomainType_DOMAIN_TYPE_ORGANIZATION
	case authorization.DomainCanvas:
		return pb.DomainType_DOMAIN_TYPE_CANVAS
	default:
		return pb.DomainType_DOMAIN_TYPE_UNSPECIFIED
	}
}
