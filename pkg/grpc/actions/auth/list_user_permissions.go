package auth

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListUserPermissions(ctx context.Context, domainType string, domainID string, userID string, authService authorization.Authorization) (*pb.ListUserPermissionsResponse, error) {
	var roles []*authorization.RoleDefinition
	var err error
	switch domainType {
	case models.DomainTypeOrganization:
		roles, err = authService.GetUserRolesForOrg(userID, domainID)
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
				DomainType: actions.DomainTypeToProto(domainType),
			}
		}
	}

	permissions := make([]*pbAuth.Permission, 0, len(permissionSet))
	for _, perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	return &pb.ListUserPermissionsResponse{
		UserId:      userID,
		DomainType:  actions.DomainTypeToProto(domainType),
		DomainId:    domainID,
		Permissions: permissions,
	}, nil
}
