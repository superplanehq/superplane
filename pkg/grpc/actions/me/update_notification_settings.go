package me

import (
	"context"

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

	workspaces := requested.GetWorkspaces()
	if workspaces == nil {
		return nil, grpcerrors.InvalidArgument(nil, "workspaces is required")
	}

	scope, ok := notificationScopeFromProto(workspaces.GetScope())
	if !ok {
		return nil, grpcerrors.InvalidArgument(nil, "workspace scope must be all, filtered, or none")
	}

	params := models.UserNotificationSettingsParams{
		WorkspaceScope: scope,
	}

	var settings *models.UserNotificationSettings
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if scope == models.NotificationWorkspaceScopeFiltered {
			filters, resolveErr := resolveWorkspaceFilters(tx, orgID, workspaces.GetFilters())
			if resolveErr != nil {
				return resolveErr
			}
			if len(filters) == 0 {
				return grpcerrors.InvalidArgument(nil, "select at least one workspace or use the all workspaces scope")
			}
			params.WorkspaceFilters = filters
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

func mapNotificationSettingsError(err error) error {
	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}
	return grpcerrors.Internal(err, "failed to update notification settings")
}
