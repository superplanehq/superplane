package auth

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
)

func UpdateRole(ctx context.Context, domainType string, domainID string, roleName string, roleSpec *pb.Role_Spec, authService authorization.Authorization) (*pb.UpdateRoleResponse, error) {
	requesterID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if roleName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "role name must be specified")
	}

	if roleSpec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "role spec must be specified")
	}

	_, err := authService.GetRoleDefinition(ctx, roleName, domainType, domainID)
	if err != nil {
		log.Errorf("role %s not found: %v", roleName, err)
		return nil, grpcerrors.NotFound(err, "role not found")
	}

	permissions := []*authorization.Permission{}
	if roleSpec.Permissions != nil {
		for _, perm := range roleSpec.Permissions {
			if perm == nil {
				continue
			}
			permissions = append(permissions, &authorization.Permission{
				Resource:   perm.Resource,
				Action:     perm.Action,
				DomainType: domainType,
			})
		}
	}

	for _, permission := range permissions {
		if !authService.IsValidPermission(domainType, permission) {
			return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("invalid permission: %s %s", permission.Resource, permission.Action))
		}
	}

	if err := ensureCanGrantPermissions(ctx, authService, requesterID, domainID, permissions); err != nil {
		return nil, err
	}

	var displayName, description string
	if roleSpec.DisplayName != "" || roleSpec.Description != "" {
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
		inheritedName := roleSpec.InheritedRole.Metadata.Name
		inheritedRoleDef, err := authService.GetRoleDefinition(ctx, inheritedName, domainType, domainID)
		if err != nil {
			log.Errorf("failed to get inherited role %s: %v", inheritedName, err)
			return nil, grpcerrors.InvalidArgument(nil, "inherited role not found")
		}
		if err := ensureCanGrantRole(ctx, authService, requesterID, domainType, domainID, inheritedName); err != nil {
			return nil, err
		}
		roleDefinition.InheritsFrom = inheritedRoleDef
	}

	err = authService.UpdateCustomRole(domainID, roleDefinition)
	if err != nil {
		log.Errorf("failed to update role %s: %v", roleName, err)
		return nil, grpcerrors.Internal(err, "failed to update role")
	}

	log.Infof("updated custom role %s in domain %s (%s)", roleName, domainID, domainType)

	return &pb.UpdateRoleResponse{}, nil
}
