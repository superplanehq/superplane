package me

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetUser(ctx context.Context, authService authorization.Authorization, includePermissions bool) (*pb.MeResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	user, err := models.FindActiveUserByID(orgID, userID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	userProto := &pb.User{
		Id:             user.ID.String(),
		Email:          user.GetEmail(),
		OrganizationId: orgID,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		HasToken:       user.TokenHash != "",
		Permissions:    []*pbAuth.Permission{},
		Roles:          []string{},
	}

	if !includePermissions {
		return &pb.MeResponse{
			User: userProto,
		}, nil
	}

	roles, err := authService.GetUserRolesForOrg(userID, user.OrganizationID.String())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	//
	// NOTE: authService.GetUserRolesForOrg returns implicit roles,
	// so when serializing, we need to make sure permissions are only added once.
	//
	permissionSet := make(map[string]*pbAuth.Permission)
	for _, role := range roles {
		userProto.Roles = append(userProto.Roles, role.Name)
		for _, permission := range role.Permissions {
			key := fmt.Sprintf("%s:%s", permission.Resource, permission.Action)
			permissionSet[key] = &pbAuth.Permission{
				Resource:   permission.Resource,
				Action:     permission.Action,
				DomainType: actions.DomainTypeToProto(models.DomainTypeOrganization),
			}
		}
	}

	permissions := make([]*pbAuth.Permission, 0, len(permissionSet))
	for _, perm := range permissionSet {
		permissions = append(permissions, perm)
	}

	userProto.Permissions = permissions

	return &pb.MeResponse{
		User: userProto,
	}, nil
}
