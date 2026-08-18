package me

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
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
	factoryIDs := []string(settings.FactoryIDs)
	if factoryIDs == nil {
		factoryIDs = []string{}
	}

	return &pb.NotificationSettings{
		Enabled:                 settings.Enabled,
		WorkspaceScope:          notificationScopeToProto(settings.WorkspaceScope),
		FactoryIds:              factoryIDs,
		WorkOrderAssigned:       notificationTypeEnabled(settings, models.NotificationTypeWorkOrderAssigned),
		WorkOrderCommentOwned:   notificationTypeEnabled(settings, models.NotificationTypeWorkOrderCommentOwned),
		WorkOrderCommentCreated: notificationTypeEnabled(settings, models.NotificationTypeWorkOrderCommentCreated),
		WorkOrderStatusOwned:    notificationTypeEnabled(settings, models.NotificationTypeWorkOrderStatusOwned),
		WorkOrderArtifactOwned:  notificationTypeEnabled(settings, models.NotificationTypeWorkOrderArtifactOwned),
	}
}

// notificationTypeEnabled reads the raw type toggle for serialization,
// independent of the master switch (unlike NotifiesType, which is used
// at delivery time).
func notificationTypeEnabled(settings *models.UserNotificationSettings, notificationType string) bool {
	enabled, ok := settings.Types.Data()[notificationType]
	if !ok {
		return true
	}
	return enabled
}

func notificationScopeToProto(scope string) pb.NotificationSettings_WorkspaceScope {
	switch scope {
	case models.NotificationWorkspaceScopeSelected:
		return pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED
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
	case pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED:
		return models.NotificationWorkspaceScopeSelected, true
	}
	return "", false
}

func notificationFactoryIDs(scope string, requestedIDs []string) ([]uuid.UUID, error) {
	if scope != models.NotificationWorkspaceScopeSelected {
		return nil, nil
	}

	seen := map[uuid.UUID]struct{}{}
	factoryIDs := make([]uuid.UUID, 0, len(requestedIDs))
	for _, requestedID := range requestedIDs {
		factoryID, err := uuid.Parse(requestedID)
		if err != nil {
			return nil, grpcerrors.InvalidArgument(err, fmt.Sprintf("invalid workspace id %q", requestedID))
		}
		if _, duplicate := seen[factoryID]; duplicate {
			continue
		}
		seen[factoryID] = struct{}{}
		factoryIDs = append(factoryIDs, factoryID)
	}

	return factoryIDs, nil
}

func factoryIDsToStrings(factoryIDs []uuid.UUID) []string {
	result := make([]string, 0, len(factoryIDs))
	for _, factoryID := range factoryIDs {
		result = append(result, factoryID.String())
	}
	return result
}
