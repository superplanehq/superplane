package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	NotificationWorkspaceScopeAll      = "all"
	NotificationWorkspaceScopeFiltered = "filtered"
	NotificationWorkspaceScopeNone     = "none"

	NotificationTypeWorkOrderAssigned       = "work_order_assigned"
	NotificationTypeWorkOrderCommentOwned   = "work_order_comment_owned"
	NotificationTypeWorkOrderCommentCreated = "work_order_comment_created"
	NotificationTypeWorkOrderStatusOwned    = "work_order_status_owned"
	NotificationTypeWorkOrderArtifactOwned  = "work_order_artifact_owned"
)

// NotificationTypes lists every configurable notification type.
// `work_order_mention` is reserved for a future release and is
// intentionally not included.
var NotificationTypes = []string{
	NotificationTypeWorkOrderAssigned,
	NotificationTypeWorkOrderCommentOwned,
	NotificationTypeWorkOrderCommentCreated,
	NotificationTypeWorkOrderStatusOwned,
	NotificationTypeWorkOrderArtifactOwned,
}

var ErrNotificationWorkspaceScopeInvalid = errors.New("workspace scope must be all, filtered, or none")

// NotificationWorkspaceFilter selects event types for one workspace
// when the scope is filtered.
type NotificationWorkspaceFilter struct {
	WorkspaceID string   `json:"workspace_id"`
	EventTypes  []string `json:"event_types"`
}

// UserNotificationSettings holds a user's organization-wide email
// notification configuration for workspace work order activity. A user
// without a row uses DefaultUserNotificationSettings: all events from
// all workspaces.
type UserNotificationSettings struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	UserID           uuid.UUID
	WorkspaceScope   string
	WorkspaceFilters datatypes.JSONType[[]NotificationWorkspaceFilter]
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UserNotificationSettingsParams carries the caller-editable fields for
// UpsertUserNotificationSettings.
type UserNotificationSettingsParams struct {
	WorkspaceScope   string
	WorkspaceFilters []NotificationWorkspaceFilter
}

// DefaultUserNotificationSettings is the configuration SuperPlane uses
// when the user has not saved settings yet.
func DefaultUserNotificationSettings() UserNotificationSettings {
	return UserNotificationSettings{
		WorkspaceScope: NotificationWorkspaceScopeAll,
	}
}

func (UserNotificationSettings) TableName() string {
	return "user_notification_settings"
}

// Notifies reports whether the settings allow an email for the given
// workspace and notification type.
func (s *UserNotificationSettings) Notifies(workspaceID uuid.UUID, notificationType string) bool {
	switch s.WorkspaceScope {
	case NotificationWorkspaceScopeNone:
		return false
	case NotificationWorkspaceScopeFiltered:
		for _, filter := range s.WorkspaceFilters.Data() {
			if filter.WorkspaceID != workspaceID.String() {
				continue
			}
			return slices.Contains(filter.EventTypes, notificationType)
		}
		return false
	default:
		return true
	}
}

func IsValidNotificationWorkspaceScope(scope string) bool {
	return scope == NotificationWorkspaceScopeAll ||
		scope == NotificationWorkspaceScopeFiltered ||
		scope == NotificationWorkspaceScopeNone
}

func FindUserNotificationSettings(tx *gorm.DB, organizationID, userID uuid.UUID) (*UserNotificationSettings, error) {
	var settings UserNotificationSettings
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		First(&settings).
		Error
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// FindUserNotificationSettingsForUsers batch-loads settings for recipient
// resolution. Users without a row are absent from the result map.
func FindUserNotificationSettingsForUsers(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userIDs []uuid.UUID,
) (map[uuid.UUID]UserNotificationSettings, error) {
	settingsByUserID := map[uuid.UUID]UserNotificationSettings{}
	if len(userIDs) == 0 {
		return settingsByUserID, nil
	}

	var settings []UserNotificationSettings
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id IN ?", userIDs).
		Find(&settings).
		Error
	if err != nil {
		return nil, err
	}

	for _, s := range settings {
		settingsByUserID[s.UserID] = s
	}

	return settingsByUserID, nil
}

func UpsertUserNotificationSettings(
	tx *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	params UserNotificationSettingsParams,
) (*UserNotificationSettings, error) {
	if !IsValidNotificationWorkspaceScope(params.WorkspaceScope) {
		return nil, ErrNotificationWorkspaceScopeInvalid
	}

	filters := params.WorkspaceFilters
	if filters == nil {
		filters = []NotificationWorkspaceFilter{}
	}

	now := time.Now()
	settings := &UserNotificationSettings{
		ID:               uuid.New(),
		OrganizationID:   organizationID,
		UserID:           userID,
		WorkspaceScope:   params.WorkspaceScope,
		WorkspaceFilters: datatypes.NewJSONType(filters),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"workspace_scope",
				"workspace_filters",
				"updated_at",
			}),
		}).
		Create(settings).
		Error
	if err != nil {
		return nil, err
	}

	return FindUserNotificationSettings(tx, organizationID, userID)
}
