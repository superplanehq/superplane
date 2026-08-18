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

func CreateRole(ctx context.Context, domainType string, domainID string, role *pb.Role, authService authorization.Authorization) (*pb.CreateRoleResponse, error) {
	requesterID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if role == nil {
		return nil, grpcerrors.InvalidArgument(nil, "role must be specified")
	}

	if role.Metadata == nil {
		return nil, grpcerrors.InvalidArgument(nil, "role metadata must be specified")
	}

	if role.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "role spec must be specified")
	}

	if role.Metadata.Name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "role name must be specified")
	}

	if role.Spec.Permissions == nil {
		return nil, grpcerrors.InvalidArgument(nil, "role permissions must be specified")
	}

	permissions := make([]*authorization.Permission, 0, len(role.Spec.Permissions))
	for _, perm := range role.Spec.Permissions {
		if perm == nil {
			continue
		}
		permissions = append(permissions, &authorization.Permission{
			Resource:   perm.Resource,
			Action:     perm.Action,
			DomainType: domainType,
		})
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

	if role.Spec.DisplayName != "" {
		displayName = role.Spec.DisplayName
	} else {
		displayName = role.Metadata.Name
	}

	if role.Spec.Description != "" {
		description = role.Spec.Description
	}

	roleDefinition := &authorization.RoleDefinition{
		Name:        role.Metadata.Name,
		DomainType:  domainType,
		Permissions: permissions,
		DisplayName: displayName,
		Description: description,
	}

	if role.Spec.InheritedRole != nil && role.Spec.InheritedRole.Metadata != nil && role.Spec.InheritedRole.Metadata.Name != "" {
		inheritedName := role.Spec.InheritedRole.Metadata.Name
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

	err := authService.CreateCustomRole(domainID, roleDefinition)
	if err != nil {
		log.Errorf("failed to create role %s: %v", role.Metadata.Name, err)
		return nil, grpcerrors.Internal(err, "failed to create role")
	}

	log.Infof("created custom role %s in domain %s (%s)", role.Metadata.Name, domainID, domainType)

	return &pb.CreateRoleResponse{}, nil
}
