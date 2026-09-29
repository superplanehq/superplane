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

	NotificationChannelEmail   = "email"
	NotificationChannelBrowser = "browser"

	NotificationTypeWorkOrderStatusOwned     = "work_order_status_owned"
	NotificationTypeWorkOrderStatusNoteOwned = "work_order_status_note_owned"
	NotificationTypeWorkOrderAgentQuestion   = "work_order_agent_question"
	NotificationTypeWorkOrderPlanReady       = "work_order_plan_ready"

	// NotificationTypeNoneSelected is stored when an all-workspaces list has
	// no current types. Delivery ignores unknown names, so every current
	// type stays off. Serialize drops this name, so the API returns an
	// empty list rather than the empty-list default.
	NotificationTypeNoneSelected = "none"
)

// NotificationTypes lists every configurable notification type.
var NotificationTypes = []string{
	NotificationTypeWorkOrderStatusOwned,
	NotificationTypeWorkOrderStatusNoteOwned,
	NotificationTypeWorkOrderAgentQuestion,
	NotificationTypeWorkOrderPlanReady,
}

var ErrNotificationWorkspaceScopeInvalid = errors.New("workspace scope must be all, filtered, or none")

// NotificationWorkspaceFilter selects event types for one workspace
// when the scope is filtered.
type NotificationWorkspaceFilter struct {
	WorkspaceID string   `json:"workspace_id"`
	EventTypes  []string `json:"event_types"`
}

// UserNotificationSettings holds a user's organization-wide
// notification configuration for workspace work order activity. A user
// without a row uses DefaultUserNotificationSettings: email for all
// events from all workspaces, and the browser channel off.
type UserNotificationSettings struct {
	ID                      uuid.UUID
	OrganizationID          uuid.UUID
	UserID                  uuid.UUID
	WorkspaceScope          string
	WorkspaceFilters        datatypes.JSONType[[]NotificationWorkspaceFilter]
	EventTypes              datatypes.JSONType[[]string]
	BrowserWorkspaceScope   string
	BrowserWorkspaceFilters datatypes.JSONType[[]NotificationWorkspaceFilter]
	BrowserEventTypes       datatypes.JSONType[[]string]
	BrowserShowWhileViewing bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// UserNotificationSettingsParams carries the caller-editable fields for
// UpsertUserNotificationSettings.
type UserNotificationSettingsParams struct {
	WorkspaceScope          string
	WorkspaceFilters        []NotificationWorkspaceFilter
	EventTypes              []string
	BrowserWorkspaceScope   string
	BrowserWorkspaceFilters []NotificationWorkspaceFilter
	BrowserEventTypes       []string
	BrowserShowWhileViewing *bool
}

// DefaultUserNotificationSettings is the configuration SuperPlane uses
// when the user has not saved settings yet.
func DefaultUserNotificationSettings() UserNotificationSettings {
	return UserNotificationSettings{
		WorkspaceScope:          NotificationWorkspaceScopeAll,
		BrowserWorkspaceScope:   NotificationWorkspaceScopeNone,
		BrowserShowWhileViewing: true,
	}
}

func (UserNotificationSettings) TableName() string {
	return "user_notification_settings"
}

// Notifies reports whether the settings allow an email for the given
// workspace and notification type.
func (s *UserNotificationSettings) Notifies(workspaceID uuid.UUID, notificationType string) bool {
	return s.NotifiesChannel(NotificationChannelEmail, workspaceID, notificationType)
}

// NotifiesChannel reports whether the settings allow a notification on
// the given channel for the workspace and notification type.
func (s *UserNotificationSettings) NotifiesChannel(channel string, workspaceID uuid.UUID, notificationType string) bool {
	scope, filters, eventTypes := s.channelSettings(channel)
	return notifiesByScope(scope, filters, eventTypes, workspaceID, notificationType)
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

	browserScope := params.BrowserWorkspaceScope
	if browserScope == "" {
		browserScope = NotificationWorkspaceScopeNone
	}
	if !IsValidNotificationWorkspaceScope(browserScope) {
		return nil, ErrNotificationWorkspaceScopeInvalid
	}

	filters := params.WorkspaceFilters
	if filters == nil {
		filters = []NotificationWorkspaceFilter{}
	}
	eventTypes := params.EventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}
	browserFilters := params.BrowserWorkspaceFilters
	if browserFilters == nil {
		browserFilters = []NotificationWorkspaceFilter{}
	}
	browserEventTypes := params.BrowserEventTypes
	if browserEventTypes == nil {
		browserEventTypes = []string{}
	}
	showWhileViewing := true
	if params.BrowserShowWhileViewing != nil {
		showWhileViewing = *params.BrowserShowWhileViewing
	}

	now := time.Now()
	settings := &UserNotificationSettings{
		ID:                      uuid.New(),
		OrganizationID:          organizationID,
		UserID:                  userID,
		WorkspaceScope:          params.WorkspaceScope,
		WorkspaceFilters:        datatypes.NewJSONType(filters),
		EventTypes:              datatypes.NewJSONType(eventTypes),
		BrowserWorkspaceScope:   browserScope,
		BrowserWorkspaceFilters: datatypes.NewJSONType(browserFilters),
		BrowserEventTypes:       datatypes.NewJSONType(browserEventTypes),
		BrowserShowWhileViewing: showWhileViewing,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	updateColumns := []string{
		"workspace_scope",
		"workspace_filters",
		"event_types",
		"updated_at",
	}
	if updatesBrowserChannel(params) {
		updateColumns = append(updateColumns,
			"browser_workspace_scope",
			"browser_workspace_filters",
			"browser_event_types",
			"browser_show_while_viewing",
		)
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns(updateColumns),
		}).
		Create(settings).
		Error
	if err != nil {
		return nil, err
	}

	return FindUserNotificationSettings(tx, organizationID, userID)
}

func updatesBrowserChannel(params UserNotificationSettingsParams) bool {
	return params.BrowserWorkspaceScope != "" ||
		params.BrowserWorkspaceFilters != nil ||
		params.BrowserEventTypes != nil ||
		params.BrowserShowWhileViewing != nil
}

func (s *UserNotificationSettings) channelSettings(channel string) (string, []NotificationWorkspaceFilter, []string) {
	if channel == NotificationChannelBrowser {
		scope := s.BrowserWorkspaceScope
		if scope == "" {
			scope = NotificationWorkspaceScopeNone
		}
		return scope, s.BrowserWorkspaceFilters.Data(), s.BrowserEventTypes.Data()
	}

	return s.WorkspaceScope, s.WorkspaceFilters.Data(), s.EventTypes.Data()
}

func notifiesByScope(
	scope string,
	filters []NotificationWorkspaceFilter,
	eventTypes []string,
	workspaceID uuid.UUID,
	notificationType string,
) bool {
	switch scope {
	case NotificationWorkspaceScopeNone:
		return false
	case NotificationWorkspaceScopeFiltered:
		for _, filter := range filters {
			if filter.WorkspaceID != workspaceID.String() {
				continue
			}
			return slices.Contains(filter.EventTypes, notificationType)
		}
		return false
	default:
		return notifiesAllScopeType(eventTypes, notificationType)
	}
}

func notifiesAllScopeType(eventTypes []string, notificationType string) bool {
	if len(eventTypes) == 0 {
		return true
	}
	return slices.Contains(eventTypes, notificationType)
}
