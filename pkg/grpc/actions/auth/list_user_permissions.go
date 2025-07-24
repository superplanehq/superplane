package auth

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListUserPermissions(ctx context.Context, domainType string, domainID string, req *pb.ListUserPermissionsRequest, authService authorization.Authorization) (*pb.ListUserPermissionsResponse, error) {
	err := actions.ValidateUUIDs(req.UserId, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.DomainType == pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roles []*authorization.RoleDefinition
	switch req.DomainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		roles, err = authService.GetUserRolesForOrg(req.UserId, req.DomainId)
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		roles, err = authService.GetUserRolesForCanvas(req.UserId, req.DomainId)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	permissionSet := make(map[string]*pbAuth.Permission)

	for _, role := range roles {
		rolePermissions := role.Permissions

		for _, perm := range rolePermissions {
			key := fmt.Sprintf("%s:%s", perm.Resource, perm.Action)
			permissionSet[key] = &pbAuth.Permission{
				Resource:   perm.Resource,
				Action:     perm.Action,
				DomainType: req.DomainType,
			}
		}
	}

	permissions := make([]*pbAuth.Permission, 0, len(permissionSet))
	for _, perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return &pb.ListUserPermissionsResponse{
		UserId:      req.UserId,
		DomainType:  req.DomainType,
		DomainId:    req.DomainId,
		Permissions: permissions,
	}, nil
}
