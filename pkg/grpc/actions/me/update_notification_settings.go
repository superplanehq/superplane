package me

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"gorm.io/gorm"
)

func UpdateNotificationSettings(
	ctx context.Context,
	req *pb.UpdateNotificationSettingsRequest,
) (*pb.UpdateNotificationSettingsResponse, error) {
	orgID, err := currentOrganizationID(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	requested := req.GetSettings()
	if requested == nil {
		return nil, grpcerrors.InvalidArgument(nil, "settings is required")
	}

	scope, ok := notificationScopeFromProto(requested.GetWorkspaceScope())
	if !ok {
		return nil, grpcerrors.InvalidArgument(nil, "workspace scope must be all or selected")
	}

	factoryIDs, err := notificationFactoryIDs(scope, requested.GetFactoryIds())
	if err != nil {
		return nil, err
	}

	if requested.GetEnabled() && scope == models.NotificationWorkspaceScopeSelected && len(factoryIDs) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "select at least one workspace or use the all workspaces scope")
	}

	params := models.UserNotificationSettingsParams{
		Enabled:        requested.GetEnabled(),
		WorkspaceScope: scope,
		FactoryIDs:     factoryIDsToStrings(factoryIDs),
		Types: map[string]bool{
			models.NotificationTypeWorkOrderAssigned:       requested.GetWorkOrderAssigned(),
			models.NotificationTypeWorkOrderCommentOwned:   requested.GetWorkOrderCommentOwned(),
			models.NotificationTypeWorkOrderCommentCreated: requested.GetWorkOrderCommentCreated(),
			models.NotificationTypeWorkOrderStatusOwned:    requested.GetWorkOrderStatusOwned(),
			models.NotificationTypeWorkOrderArtifactOwned:  requested.GetWorkOrderArtifactOwned(),
			models.NotificationTypeWorkOrderMention:        requested.GetWorkOrderMentioned(),
		},
	}

	var settings *models.UserNotificationSettings
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureFactoriesExist(tx, orgID, factoryIDs); err != nil {
			return err
		}

		settings, err = models.UpsertUserNotificationSettings(tx, orgID, userID, params)
		return err
	})
	if err != nil {
		return nil, mapNotificationSettingsError(err)
	}

	return &pb.UpdateNotificationSettingsResponse{
		Settings: serializeNotificationSettings(settings),
	}, nil
}

func ensureFactoriesExist(tx *gorm.DB, orgID uuid.UUID, factoryIDs []uuid.UUID) error {
	if len(factoryIDs) == 0 {
		return nil
	}

	count, err := models.CountFactoriesByIDs(tx, orgID, factoryIDs)
	if err != nil {
		return err
	}

	if count != int64(len(factoryIDs)) {
		return grpcerrors.InvalidArgument(nil, "one or more selected workspaces were not found")
	}

	return nil
}

func mapNotificationSettingsError(err error) error {
	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}
	return grpcerrors.Internal(err, "failed to update notification settings")
}
