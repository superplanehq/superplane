package factories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateNotificationSettings(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateNotificationSettingsRequest,
) (*pb.UpdateNotificationSettingsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update notification settings")
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	requested := req.GetSettings()
	if requested == nil {
		return nil, factoryErrorToStatus(invalidArgument("settings is required"), "failed to update notification settings")
	}

	scope, ok := notificationScopeFromProto(requested.GetWorkspaceScope())
	if !ok {
		return nil, factoryErrorToStatus(
			invalidArgument("workspace scope must be all or selected"),
			"failed to update notification settings",
		)
	}

	factoryIDs, err := notificationFactoryIDs(scope, requested.GetFactoryIds())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update notification settings")
	}

	if requested.GetEnabled() && scope == models.NotificationWorkspaceScopeSelected && len(factoryIDs) == 0 {
		return nil, factoryErrorToStatus(
			invalidArgument("select at least one workspace or use the all workspaces scope"),
			"failed to update notification settings",
		)
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
		return nil, factoryErrorToStatus(err, "failed to update notification settings")
	}

	return &pb.UpdateNotificationSettingsResponse{
		Settings: serializeNotificationSettings(settings),
	}, nil
}

// notificationFactoryIDs parses the requested workspace list. An `all`
// scope ignores the list. An empty `selected` list is allowed when
// notifications are off; the caller rejects it when they are on.
func notificationFactoryIDs(scope string, requestedIDs []string) ([]uuid.UUID, error) {
	if scope != models.NotificationWorkspaceScopeSelected {
		return nil, nil
	}

	seen := map[uuid.UUID]struct{}{}
	factoryIDs := make([]uuid.UUID, 0, len(requestedIDs))
	for _, requestedID := range requestedIDs {
		factoryID, err := uuid.Parse(requestedID)
		if err != nil {
			return nil, invalidArgument(fmt.Sprintf("invalid workspace id %q", requestedID))
		}
		if _, duplicate := seen[factoryID]; duplicate {
			continue
		}
		seen[factoryID] = struct{}{}
		factoryIDs = append(factoryIDs, factoryID)
	}

	return factoryIDs, nil
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
		return invalidArgument("one or more selected workspaces were not found")
	}

	return nil
}

func factoryIDsToStrings(factoryIDs []uuid.UUID) []string {
	result := make([]string, 0, len(factoryIDs))
	for _, factoryID := range factoryIDs {
		result = append(result, factoryID.String())
	}
	return result
}
