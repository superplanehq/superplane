package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest, authService authorization.Authorization) (*pb.UpdateRoleResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.RoleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	domainType := convertDomainType(req.DomainType)
	if domainType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid domain type")
	}

	// Check if role exists
	_, err = authService.GetRoleDefinition(req.RoleName, domainType, req.DomainId)
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
			DomainType: convertDomainType(perm.DomainType),
		}
	}

	roleDefinition := &authorization.RoleDefinition{
		Name:        req.RoleName,
		DomainType:  domainType,
		Permissions: permissions,
	}

	// Handle inherited role if specified
	if req.InheritedRole != "" {
		inheritedRoleDef, err := authService.GetRoleDefinition(req.InheritedRole, domainType, req.DomainId)
		if err != nil {
			log.Errorf("failed to get inherited role %s: %v", req.InheritedRole, err)
			return nil, status.Error(codes.InvalidArgument, "inherited role not found")
		}
		roleDefinition.InheritsFrom = inheritedRoleDef
	}

	err = authService.UpdateCustomRole(req.DomainId, roleDefinition)
	if err != nil {
		log.Errorf("failed to update role %s: %v", req.RoleName, err)
		return nil, status.Error(codes.Internal, "failed to update role")
	}

	log.Infof("updated custom role %s in domain %s (%s)", req.RoleName, req.DomainId, domainType)

	return &pb.UpdateRoleResponse{}, nil
}