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
	NotificationWorkspaceScopeSelected = "selected"

	NotificationTypeWorkOrderAssigned       = "work_order_assigned"
	NotificationTypeWorkOrderCommentOwned   = "work_order_comment_owned"
	NotificationTypeWorkOrderCommentCreated = "work_order_comment_created"
	NotificationTypeWorkOrderStatusOwned    = "work_order_status_owned"
	NotificationTypeWorkOrderArtifactOwned  = "work_order_artifact_owned"
	NotificationTypeWorkOrderMention        = "work_order_mention"
)

// NotificationTypes lists every configurable notification type.
var NotificationTypes = []string{
	NotificationTypeWorkOrderAssigned,
	NotificationTypeWorkOrderCommentOwned,
	NotificationTypeWorkOrderCommentCreated,
	NotificationTypeWorkOrderStatusOwned,
	NotificationTypeWorkOrderArtifactOwned,
	NotificationTypeWorkOrderMention,
}

var ErrNotificationWorkspaceScopeInvalid = errors.New("workspace scope must be all or selected")

// UserNotificationSettings holds a user's organization-wide email
// notification configuration for factory work order activity. A user
// without a row uses DefaultUserNotificationSettings: emails on, all
// types, all workspaces.
type UserNotificationSettings struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Enabled        bool
	WorkspaceScope string
	FactoryIDs     datatypes.JSONSlice[string]
	// Types maps a notification type to its toggle. A missing key means
	// the type is on, so newly introduced types default to enabled.
	Types     datatypes.JSONType[map[string]bool]
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserNotificationSettingsParams carries the caller-editable fields for
// UpsertUserNotificationSettings.
type UserNotificationSettingsParams struct {
	Enabled        bool
	WorkspaceScope string
	FactoryIDs     []string
	Types          map[string]bool
}

// DefaultUserNotificationSettings is the configuration SuperPlane uses
// when the user has not saved settings yet.
func DefaultUserNotificationSettings() UserNotificationSettings {
	return UserNotificationSettings{
		Enabled:        true,
		WorkspaceScope: NotificationWorkspaceScopeAll,
	}
}

func (UserNotificationSettings) TableName() string {
	return "user_notification_settings"
}

// NotifiesType reports whether the settings allow emails for the given
// notification type. A missing key defaults to on.
func (s *UserNotificationSettings) NotifiesType(notificationType string) bool {
	if !s.Enabled {
		return false
	}

	enabled, ok := s.Types.Data()[notificationType]
	if !ok {
		return true
	}
	return enabled
}

// AppliesToFactory reports whether the settings cover activity in the
// given factory (workspace).
func (s *UserNotificationSettings) AppliesToFactory(factoryID uuid.UUID) bool {
	if s.WorkspaceScope != NotificationWorkspaceScopeSelected {
		return true
	}
	return slices.Contains(s.FactoryIDs, factoryID.String())
}

func IsValidNotificationWorkspaceScope(scope string) bool {
	return scope == NotificationWorkspaceScopeAll || scope == NotificationWorkspaceScopeSelected
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

	factoryIDs := params.FactoryIDs
	if factoryIDs == nil {
		factoryIDs = []string{}
	}
	types := params.Types
	if types == nil {
		types = map[string]bool{}
	}

	now := time.Now()
	settings := &UserNotificationSettings{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		UserID:         userID,
		Enabled:        params.Enabled,
		WorkspaceScope: params.WorkspaceScope,
		FactoryIDs:     datatypes.NewJSONSlice(factoryIDs),
		Types:          datatypes.NewJSONType(types),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"enabled",
				"workspace_scope",
				"factory_ids",
				"types",
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
