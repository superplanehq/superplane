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
		Enabled:        settings.Enabled,
		WorkspaceScope: notificationScopeToProto(settings.WorkspaceScope),
		FactoryIds:     factoryIDs,
		Types:          serializeNotificationTypes(settings),
	}
}

func serializeNotificationTypes(settings *models.UserNotificationSettings) []*pb.NotificationSettings_TypeToggle {
	toggles := make([]*pb.NotificationSettings_TypeToggle, 0, len(models.NotificationTypes))
	for _, notificationType := range models.NotificationTypes {
		protoType, ok := notificationTypeToProto(notificationType)
		if !ok {
			continue
		}
		toggles = append(toggles, &pb.NotificationSettings_TypeToggle{
			Type:    protoType,
			Enabled: notificationTypeEnabled(settings, notificationType),
		})
	}
	return toggles
}

func notificationTypesFromProto(toggles []*pb.NotificationSettings_TypeToggle) (map[string]bool, error) {
	types := map[string]bool{}
	for _, toggle := range toggles {
		if toggle == nil {
			continue
		}
		if toggle.GetType() == pb.NotificationSettings_TYPE_UNSPECIFIED {
			return nil, grpcerrors.InvalidArgument(nil, "notification type is required")
		}
		name, ok := notificationTypeFromProto(toggle.GetType())
		if !ok {
			return nil, grpcerrors.InvalidArgument(nil, "unknown notification type")
		}
		types[name] = toggle.GetEnabled()
	}
	return types, nil
}

var notificationTypeProto = map[string]pb.NotificationSettings_Type{
	models.NotificationTypeWorkOrderAssigned:       pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED,
	models.NotificationTypeWorkOrderCommentOwned:   pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED,
	models.NotificationTypeWorkOrderCommentCreated: pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_CREATED,
	models.NotificationTypeWorkOrderStatusOwned:    pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
	models.NotificationTypeWorkOrderArtifactOwned:  pb.NotificationSettings_TYPE_WORK_ORDER_ARTIFACT_OWNED,
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
