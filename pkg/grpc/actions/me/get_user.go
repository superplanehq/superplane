package me

import (
	"context"
	"errors"

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
	err = telemetry.RunSpan(ctx, "auth.load_user_roles", func(ctx context.Context) error {
		var loadErr error
		roles, loadErr = authService.GetUserRolesForOrg(ctx, userID, user.OrganizationID.String())
		return loadErr
	})
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
	err = telemetry.RunSpan(ctx, "auth.load_user_groups", func(ctx context.Context) error {
		var loadErr error
		groups, loadErr = authService.GetUserGroups(ctx, user.OrganizationID.String(), models.DomainTypeOrganization, userID)
		return loadErr
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get user groups")
	}

	userProto.Groups = groups

	return &pb.MeResponse{
		User: userProto,
	}, nil
}

func loadUser(ctx context.Context, orgID, userID string) (*models.User, error) {
	var user *models.User
	err := telemetry.RunSpan(ctx, "auth.load_user", func(ctx context.Context) error {
		var loadErr error
		user, loadErr = models.FindActiveUserByIDInTransaction(database.DB(ctx), orgID, userID)
		return loadErr
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "user not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load user")
	}

	return user, nil
}
