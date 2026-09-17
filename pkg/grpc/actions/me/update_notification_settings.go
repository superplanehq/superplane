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

	workspaces := requested.GetWorkspaces()
	if workspaces == nil {
		return nil, grpcerrors.InvalidArgument(nil, "workspaces is required")
	}

	emailChannel, err := parseNotificationChannel(
		workspaces.GetScope(),
		workspaces.GetEventTypes(),
		workspaces.GetFilters(),
	)
	if err != nil {
		return nil, err
	}

	params := models.UserNotificationSettingsParams{
		WorkspaceScope: emailChannel.scope,
		EventTypes:     emailChannel.eventTypes,
	}

	browser := requested.GetBrowser()
	var browserChannel *parsedNotificationChannel
	if browser != nil {
		parsed, parseErr := parseNotificationChannel(
			browser.GetScope(),
			browser.GetEventTypes(),
			browser.GetFilters(),
		)
		if parseErr != nil {
			return nil, parseErr
		}
		showWhileViewing := browser.GetShowWhileViewing()
		parsed.showWhileViewing = &showWhileViewing
		browserChannel = &parsed
		params.BrowserWorkspaceScope = parsed.scope
		params.BrowserEventTypes = parsed.eventTypes
		params.BrowserShowWhileViewing = parsed.showWhileViewing
	}

	var settings *models.UserNotificationSettings
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if emailChannel.scope == models.NotificationWorkspaceScopeFiltered {
			filters, resolveErr := resolveRequiredWorkspaceFilters(tx, orgID, emailChannel.filters)
			if resolveErr != nil {
				return resolveErr
			}
			params.WorkspaceFilters = filters
		}

		if browserChannel != nil && browserChannel.scope == models.NotificationWorkspaceScopeFiltered {
			filters, resolveErr := resolveRequiredWorkspaceFilters(tx, orgID, browserChannel.filters)
			if resolveErr != nil {
				return resolveErr
			}
			params.BrowserWorkspaceFilters = filters
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

type parsedNotificationChannel struct {
	scope            string
	eventTypes       []string
	filters          []*pb.NotificationSettings_WorkspaceFilter
	showWhileViewing *bool
}

func parseNotificationChannel(
	scope pb.NotificationSettings_WorkspaceScope,
	eventTypes []pb.NotificationSettings_Type,
	filters []*pb.NotificationSettings_WorkspaceFilter,
) (parsedNotificationChannel, error) {
	parsedScope, ok := notificationScopeFromProto(scope)
	if !ok {
		return parsedNotificationChannel{}, grpcerrors.InvalidArgument(nil, "workspace scope must be all, filtered, or none")
	}

	parsed := parsedNotificationChannel{
		scope:   parsedScope,
		filters: filters,
	}
	if parsedScope == models.NotificationWorkspaceScopeAll {
		names, err := notificationTypesFromProto(eventTypes)
		if err != nil {
			return parsedNotificationChannel{}, err
		}
		parsed.eventTypes = names
	}

	return parsed, nil
}

func resolveRequiredWorkspaceFilters(
	tx *gorm.DB,
	orgID uuid.UUID,
	filters []*pb.NotificationSettings_WorkspaceFilter,
) ([]models.NotificationWorkspaceFilter, error) {
	resolved, err := resolveWorkspaceFilters(tx, orgID, filters)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, grpcerrors.InvalidArgument(nil, "select at least one workspace or use the all workspaces scope")
	}
	return resolved, nil
}

func mapNotificationSettingsError(err error) error {
	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}
	return grpcerrors.Internal(err, "failed to update notification settings")
}
