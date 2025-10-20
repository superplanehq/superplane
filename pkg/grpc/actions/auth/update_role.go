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

func UpdateRole(ctx context.Context, domainType string, domainID string, roleName string, roleSpec *pb.Role_Spec, authService authorization.Authorization) (*pb.UpdateRoleResponse, error) {
	if roleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	if authService.IsDefaultRole(roleName, domainType) {
		return nil, status.Error(codes.InvalidArgument, "cannot update default role")
	}

	if domainType == models.DomainTypeCanvas && domainID == "*" {
		orgID, ok := authentication.GetOrganizationIdFromMetadata(ctx)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "organization context required for global canvas roles")
		}

		_, err := authService.GetGlobalCanvasRoleDefinition(roleName, orgID)
		if err != nil {
			log.Errorf("global canvas role %s not found: %v", roleName, err)
			return nil, status.Error(codes.NotFound, "role not found")
		}
	} else {
		_, err := authService.GetRoleDefinition(roleName, domainType, domainID)
		if err != nil {
			log.Errorf("role %s not found: %v", roleName, err)
			return nil, status.Error(codes.NotFound, "role not found")
		}
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

	//
	// Update the role
	//
	var err error
	if domainType == models.DomainTypeCanvas && domainID == "*" {
		orgID, _ := authentication.GetOrganizationIdFromMetadata(ctx)
		err = authService.UpdateCustomRoleWithOrgContext(domainID, orgID, roleDefinition)
	} else {
		err = authService.UpdateCustomRole(domainID, roleDefinition)
	}

	if err != nil {
		log.Errorf("failed to update role %s: %v", roleName, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Infof("updated custom role %s in domain %s (%s)", roleName, domainID, domainType)

	return &pb.UpdateRoleResponse{}, nil
}
