package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateRole(ctx context.Context, domainType string, domainID string, roleName string, roleSpec *pb.Role_Spec, authService authorization.Authorization) (*pb.UpdateRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	_, err := authService.GetRoleDefinition(roleName, domainType, domainID)
	if err != nil {
		log.Errorf("role %s not found: %v", roleName, err)
		return nil, status.Error(codes.NotFound, "role not found")
	}

	permissions := []*authorization.Permission{}
	if roleSpec != nil && roleSpec.Permissions != nil {
		for _, perm := range roleSpec.Permissions {
			permissions = append(permissions, &authorization.Permission{
				Resource:   perm.Resource,
				Action:     perm.Action,
				DomainType: domainType,
			})
		}
	}

	var displayName, description string
	if roleSpec != nil && (roleSpec.DisplayName != "" || roleSpec.Description != "") {
		displayName = roleSpec.DisplayName
		if displayName == "" {
			displayName = roleName
		}
		description = roleSpec.Description
	}

	roleDefinition := &authorization.RoleDefinition{
		Name:        roleName,
		DomainType:  domainType,
		Permissions: permissions,
		DisplayName: displayName,
		Description: description,
	}

	if roleSpec.InheritedRole != nil && roleSpec.InheritedRole.Metadata != nil && roleSpec.InheritedRole.Metadata.Name != "" {
		inheritedRoleDef, err := authService.GetRoleDefinition(roleSpec.InheritedRole.Metadata.Name, domainType, domainID)
		if err != nil {
			log.Errorf("failed to get inherited role %s: %v", roleSpec.InheritedRole.Metadata.Name, err)
			return nil, status.Error(codes.InvalidArgument, "inherited role not found")
		}
		roleDefinition.InheritsFrom = inheritedRoleDef
	}

	err = authService.UpdateCustomRole(domainID, roleDefinition)
	if err != nil {
		log.Errorf("failed to update role %s: %v", roleName, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Infof("updated custom role %s in domain %s (%s)", roleName, domainID, domainType)

	return &pb.UpdateRoleResponse{}, nil
}
