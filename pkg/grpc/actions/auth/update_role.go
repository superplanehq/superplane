package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/roles"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateRole(ctx context.Context, domainType string, domainID string, req *pb.UpdateRoleRequest, authService authorization.Authorization) (*pb.UpdateRoleResponse, error) {
	if req.RoleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}


	// Check if role exists
	_, err := authService.GetRoleDefinition(req.RoleName, domainType, domainID)
	if err != nil {
		log.Errorf("role %s not found: %v", req.RoleName, err)
		return nil, status.Error(codes.NotFound, "role not found")
	}

	// Convert protobuf permissions to authorization permissions
	permissions := make([]*authorization.Permission, len(req.Permissions))
	for i, perm := range req.Permissions {
		permissions[i] = &authorization.Permission{
			Resource:   perm.Resource,
			Action:     perm.Action,
			DomainType: domainType,
		}
	}

	roleDefinition := &authorization.RoleDefinition{
		Name:        req.RoleName,
		DomainType:  domainType,
		Permissions: permissions,
	}

	// Handle inherited role if specified
	if req.InheritedRole != "" {
		inheritedRoleDef, err := authService.GetRoleDefinition(req.InheritedRole, domainType, domainID)
		if err != nil {
			log.Errorf("failed to get inherited role %s: %v", req.InheritedRole, err)
			return nil, status.Error(codes.InvalidArgument, "inherited role not found")
		}
		roleDefinition.InheritsFrom = inheritedRoleDef
	}

	err = authService.UpdateCustomRole(domainID, roleDefinition)
	if err != nil {
		log.Errorf("failed to update role %s: %v", req.RoleName, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Update role metadata if display name or description is provided
	if req.DisplayName != "" || req.Description != "" {
		displayName := req.DisplayName
		if displayName == "" {
			displayName = req.RoleName // Fallback to role name
		}

		err = models.UpsertRoleMetadata(req.RoleName, domainType, domainID, displayName, req.Description)
		if err != nil {
			log.Errorf("failed to update role metadata for %s: %v", req.RoleName, err)
			return nil, status.Error(codes.Internal, "failed to update role metadata")
		}
	}

	log.Infof("updated custom role %s in domain %s (%s)", req.RoleName, domainID, domainType)

	return &pb.UpdateRoleResponse{}, nil
}
