package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// currentUserID resolves the authenticated caller for the self-scoped
// notification settings endpoints.
func currentUserID(ctx context.Context) (uuid.UUID, error) {
	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid user id")
	}

	return userID, nil
}

func defaultNotificationSettingsProto() *pb.NotificationSettings {
	return &pb.NotificationSettings{
		Enabled:                 false,
		WorkspaceScope:          pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
		FactoryIds:              []string{},
		WorkOrderAssigned:       true,
		WorkOrderCommentOwned:   true,
		WorkOrderCommentCreated: true,
		WorkOrderStatusOwned:    true,
		WorkOrderArtifactOwned:  true,
	}
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
