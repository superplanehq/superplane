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

func CreateRole(ctx context.Context, domainType string, domainID string, req *pb.CreateRoleRequest, authService authorization.Authorization) (*pb.CreateRoleResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
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
		Name:        req.Name,
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

	err := authService.CreateCustomRole(domainID, roleDefinition)
	if err != nil {
		log.Errorf("failed to create role %s: %v", req.Name, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Create or update role metadata if display name or description is provided
	if req.DisplayName != "" || req.Description != "" {
		displayName := req.DisplayName

		if displayName == "" {
			displayName = req.Name // Fallback to role name
		}

		err = models.UpsertRoleMetadata(req.Name, domainType, domainID, displayName, req.Description)
		if err != nil {
			log.Errorf("failed to create role metadata for %s: %v", req.Name, err)
			return nil, status.Error(codes.Internal, "failed to create role metadata")
		}
	}

	log.Infof("created custom role %s in domain %s (%s)", req.Name, domainID, domainType)

	return &pb.CreateRoleResponse{}, nil
}
