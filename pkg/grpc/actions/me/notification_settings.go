package me

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"gorm.io/gorm"
)

func currentUserID(ctx context.Context) (uuid.UUID, error) {
	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid user id")
	}

	return userID, nil
}

func currentOrganizationID(ctx context.Context) (uuid.UUID, error) {
	orgIDStr, ok := authentication.GetOrganizationIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	return orgID, nil
}

func defaultNotificationSettingsProto() *pb.NotificationSettings {
	settings := models.DefaultUserNotificationSettings()
	return serializeNotificationSettings(&settings)
}

func serializeNotificationSettings(settings *models.UserNotificationSettings) *pb.NotificationSettings {
	filters := []*pb.NotificationSettings_WorkspaceFilter{}
	if settings.WorkspaceScope == models.NotificationWorkspaceScopeFiltered {
		for _, filter := range settings.WorkspaceFilters.Data() {
			filters = append(filters, &pb.NotificationSettings_WorkspaceFilter{
				WorkspaceId: filter.WorkspaceID,
				EventTypes:  serializeEventTypes(filter.EventTypes),
			})
		}
	}

	var eventTypes []pb.NotificationSettings_Type
	if settings.WorkspaceScope == models.NotificationWorkspaceScopeAll {
		eventTypes = serializeEventTypes(settings.EventTypes.Data())
	}

	return &pb.NotificationSettings{
		Workspaces: &pb.NotificationSettings_Workspaces{
			Scope:      notificationScopeToProto(settings.WorkspaceScope),
			Filters:    filters,
			EventTypes: eventTypes,
		},
	}
}

func serializeEventTypes(eventTypes []string) []pb.NotificationSettings_Type {
	protoTypes := make([]pb.NotificationSettings_Type, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		protoType, ok := notificationTypeToProto(eventType)
		if !ok {
			continue
		}
		protoTypes = append(protoTypes, protoType)
	}
	return protoTypes
}

func notificationTypesFromProto(eventTypes []pb.NotificationSettings_Type) ([]string, error) {
	names := make([]string, 0, len(eventTypes))
	seen := map[string]struct{}{}
	for _, protoType := range eventTypes {
		if protoType == pb.NotificationSettings_TYPE_UNSPECIFIED {
			return nil, grpcerrors.InvalidArgument(nil, "notification type is required")
		}
		name, ok := notificationTypeFromProto(protoType)
		if !ok {
			return nil, grpcerrors.InvalidArgument(nil, "unknown notification type")
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names, nil
}

var notificationTypeProto = map[string]pb.NotificationSettings_Type{
	models.NotificationTypeWorkOrderAssigned:       pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED,
	models.NotificationTypeWorkOrderCommentOwned:   pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED,
	models.NotificationTypeWorkOrderCommentCreated: pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_CREATED,
	models.NotificationTypeWorkOrderStatusOwned:    pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
	models.NotificationTypeWorkOrderArtifactOwned:  pb.NotificationSettings_TYPE_WORK_ORDER_ARTIFACT_OWNED,
	models.NotificationTypeWorkOrderMention:        pb.NotificationSettings_TYPE_WORK_ORDER_MENTIONED,
}

func notificationTypeToProto(notificationType string) (pb.NotificationSettings_Type, bool) {
	protoType, ok := notificationTypeProto[notificationType]
	return protoType, ok
}

func notificationTypeFromProto(protoType pb.NotificationSettings_Type) (string, bool) {
	for name, mapped := range notificationTypeProto {
		if mapped == protoType {
			return name, true
		}
	}
	return "", false
}

func notificationScopeToProto(scope string) pb.NotificationSettings_WorkspaceScope {
	switch scope {
	case models.NotificationWorkspaceScopeFiltered:
		return pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED
	case models.NotificationWorkspaceScopeNone:
		return pb.NotificationSettings_WORKSPACE_SCOPE_NONE
	case models.NotificationWorkspaceScopeAll:
		return pb.NotificationSettings_WORKSPACE_SCOPE_ALL
	default:
		return pb.NotificationSettings_WORKSPACE_SCOPE_UNSPECIFIED
	}
}

func notificationScopeFromProto(scope pb.NotificationSettings_WorkspaceScope) (string, bool) {
	switch scope {
	case pb.NotificationSettings_WORKSPACE_SCOPE_ALL:
		return models.NotificationWorkspaceScopeAll, true
	case pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED:
		return models.NotificationWorkspaceScopeFiltered, true
	case pb.NotificationSettings_WORKSPACE_SCOPE_NONE:
		return models.NotificationWorkspaceScopeNone, true
	}
	return "", false
}

func resolveWorkspaceFilters(
	tx *gorm.DB,
	orgID uuid.UUID,
	filters []*pb.NotificationSettings_WorkspaceFilter,
) ([]models.NotificationWorkspaceFilter, error) {
	seen := map[uuid.UUID]struct{}{}
	resolved := make([]models.NotificationWorkspaceFilter, 0, len(filters))

	for _, filter := range filters {
		if filter == nil {
			continue
		}

		factory, err := resolveWorkspace(tx, orgID, filter.GetWorkspaceId(), filter.GetWorkspaceKey())
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[factory.ID]; duplicate {
			continue
		}
		seen[factory.ID] = struct{}{}

		eventTypes, err := notificationTypesFromProto(filter.GetEventTypes())
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, models.NotificationWorkspaceFilter{
			WorkspaceID: factory.ID.String(),
			EventTypes:  eventTypes,
		})
	}

	return resolved, nil
}

func resolveWorkspace(tx *gorm.DB, orgID uuid.UUID, workspaceID, workspaceKey string) (*models.Factory, error) {
	if workspaceID != "" {
		factoryID, err := uuid.Parse(workspaceID)
		if err != nil {
			return nil, grpcerrors.InvalidArgument(err, fmt.Sprintf("invalid workspace id %q", workspaceID))
		}

		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return nil, mapWorkspaceLookupError(err)
		}
		return factory, nil
	}

	if workspaceKey != "" {
		factory, err := models.FindFactoryByKey(tx, orgID, workspaceKey)
		if err != nil {
			return nil, mapWorkspaceLookupError(err)
		}
		return factory, nil
	}

	return nil, grpcerrors.InvalidArgument(nil, "workspace id or workspace key is required")
}

func mapWorkspaceLookupError(err error) error {
	switch {
	case errors.Is(err, models.ErrFactoryNotFound):
		return grpcerrors.InvalidArgument(nil, "one or more selected workspaces were not found")
	case errors.Is(err, models.ErrFactoryKeyRequired):
		return grpcerrors.InvalidArgument(err, "workspace key is required")
	case errors.Is(err, models.ErrFactoryKeyInvalid):
		return grpcerrors.InvalidArgument(err, "workspace key must be 2 to 5 uppercase letters")
	}
	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}
	return err
}
