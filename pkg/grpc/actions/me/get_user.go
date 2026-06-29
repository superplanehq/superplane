package me

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func GetUser(ctx context.Context, authService authorization.Authorization, includePermissions bool) (*pb.MeResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	user, err := loadUser(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	userProto := &pb.User{
		Id:             user.ID.String(),
		Name:           user.Name,
		Email:          user.GetEmail(),
		OrganizationId: orgID,
		CreatedAt:      timestamppb.New(user.CreatedAt),
		HasToken:       user.TokenHash != "",
		Permissions:    []*pbAuth.Permission{},
		Roles:          []string{},
		Groups:         []string{},
	}

	if !includePermissions {
		return &pb.MeResponse{
			User: userProto,
		}, nil
	}

	var roles []*authorization.RoleDefinition
	roles, err = loadUserRoles(ctx, authService, userID, user.OrganizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get user roles")
	}

	//
	// NOTE: authService.GetUserRolesForOrg returns implicit roles,
	// so when serializing, we need to make sure permissions are only added once.
	//
	permissionSet := make(map[string]*pbAuth.Permission)
	for _, role := range roles {
		userProto.Roles = append(userProto.Roles, role.Name)
		for _, permission := range role.Permissions {
			key := permission.Resource + ":" + permission.Action
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

	var groups []string
	groups, err = loadUserGroups(ctx, authService, userID, user.OrganizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get user groups")
	}

	userProto.Groups = groups

	return &pb.MeResponse{
		User: userProto,
	}, nil
}

func loadUser(ctx context.Context, orgID, userID string) (user *models.User, err error) {
	ctx, done := telemetry.Span(ctx, "auth.load_user")
	defer done(&err)

	user, err = models.FindActiveUserByIDInTransaction(database.DB(ctx), orgID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "user not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load user")
	}

	return user, nil
}

func loadUserRoles(ctx context.Context, authService authorization.Authorization, userID string, organizationID uuid.UUID) (roles []*authorization.RoleDefinition, err error) {
	ctx, done := telemetry.Span(ctx, "auth.load_user_roles")
	defer done(&err)

	return authService.GetUserRolesForOrg(ctx, userID, organizationID.String())
}

func loadUserGroups(ctx context.Context, authService authorization.Authorization, userID string, organizationID uuid.UUID) (groups []string, err error) {
	ctx, done := telemetry.Span(ctx, "auth.load_user_groups")
	defer done(&err)

	return authService.GetUserGroups(ctx, organizationID.String(), models.DomainTypeOrganization, userID)
}
