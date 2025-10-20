package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateRole(ctx context.Context, domainType string, domainID string, role *pb.Role, authService authorization.Authorization) (*pb.CreateRoleResponse, error) {
	if role == nil {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	if role.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "role metadata must be specified")
	}

	if role.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "role spec must be specified")
	}

	if role.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	if role.Spec.Permissions == nil {
		return nil, status.Error(codes.InvalidArgument, "role permissions must be specified")
	}

	permissions := make([]*authorization.Permission, len(role.Spec.Permissions))
	for i, perm := range role.Spec.Permissions {
		permissions[i] = &authorization.Permission{
			Resource:   perm.Resource,
			Action:     perm.Action,
			DomainType: domainType,
		}
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
		inheritedRoleDef, err := authService.GetRoleDefinition(role.Spec.InheritedRole.Metadata.Name, domainType, domainID)
		if err != nil {
			log.Errorf("failed to get inherited role %s: %v", role.Spec.InheritedRole.Metadata.Name, err)
			return nil, status.Error(codes.InvalidArgument, "inherited role not found")
		}
		roleDefinition.InheritsFrom = inheritedRoleDef
	}

	var err error
	if domainType == models.DomainTypeCanvas && domainID == "*" {
		if orgID, ok := authentication.GetOrganizationIdFromMetadata(ctx); ok {
			err = authService.CreateCustomRoleWithOrgContext(domainID, orgID, roleDefinition)
		} else {
			return nil, status.Error(codes.InvalidArgument, "organization context required for global canvas roles")
		}
	} else {
		err = authService.CreateCustomRole(domainID, roleDefinition)
	}

	if err != nil {
		log.Errorf("failed to create role %s: %v", role.Metadata.Name, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Infof("created custom role %s in domain %s (%s)", role.Metadata.Name, domainID, domainType)

	return &pb.CreateRoleResponse{}, nil
}
