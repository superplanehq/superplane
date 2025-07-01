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

func CreateRole(ctx context.Context, req *pb.CreateRoleRequest, authService authorization.Authorization) (*pb.CreateRoleResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "role name must be specified")
	}

	domainType := convertDomainType(req.DomainType)
	if domainType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid domain type")
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
		Name:        req.Name,
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

	err = authService.CreateCustomRole(req.DomainId, roleDefinition)
	if err != nil {
		log.Errorf("failed to create role %s: %v", req.Name, err)
		return nil, status.Error(codes.Internal, "failed to create role")
	}

	log.Infof("created custom role %s in domain %s (%s)", req.Name, req.DomainId, domainType)

	return &pb.CreateRoleResponse{}, nil
}