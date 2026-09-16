package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__UserNotificationSettings(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	t.Run("find without a row -> record not found", func(t *testing.T) {
		_, err := models.FindUserNotificationSettings(db, r.Organization.ID, uuid.New())
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("upsert creates and then updates the same row", func(t *testing.T) {
		userID := support.CreateUser(t, r, r.Organization.ID).ID

		created, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     []string{models.NotificationTypeWorkOrderAssigned},
		})
		require.NoError(t, err)
		assert.Equal(t, models.NotificationWorkspaceScopeAll, created.WorkspaceScope)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderAssigned}, created.EventTypes.Data())

		factoryID := uuid.New().String()
		updated, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: factoryID,
				EventTypes:  []string{models.NotificationTypeWorkOrderAssigned},
			}},
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, models.NotificationWorkspaceScopeFiltered, updated.WorkspaceScope)
		assert.Empty(t, updated.EventTypes.Data())
		require.Len(t, updated.WorkspaceFilters.Data(), 1)
		assert.Equal(t, factoryID, updated.WorkspaceFilters.Data()[0].WorkspaceID)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderAssigned}, updated.WorkspaceFilters.Data()[0].EventTypes)
		assert.Equal(t, models.NotificationWorkspaceScopeNone, updated.BrowserWorkspaceScope)
		assert.True(t, updated.BrowserShowWhileViewing)
	})

	t.Run("email-only upsert keeps stored browser fields", func(t *testing.T) {
		userID := support.CreateUser(t, r, r.Organization.ID).ID
		showWhileViewing := false
		created, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope:          models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope:   models.NotificationWorkspaceScopeAll,
			BrowserEventTypes:       []string{models.NotificationTypeWorkOrderCommentOwned},
			BrowserShowWhileViewing: &showWhileViewing,
		})
		require.NoError(t, err)

		updated, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     []string{models.NotificationTypeWorkOrderAssigned},
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, models.NotificationWorkspaceScopeAll, updated.WorkspaceScope)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderAssigned}, updated.EventTypes.Data())
		assert.Equal(t, models.NotificationWorkspaceScopeAll, updated.BrowserWorkspaceScope)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderCommentOwned}, updated.BrowserEventTypes.Data())
		assert.False(t, updated.BrowserShowWhileViewing)
	})

	t.Run("upsert round-trips browser channel fields", func(t *testing.T) {
		userID := support.CreateUser(t, r, r.Organization.ID).ID
		factoryID := uuid.New().String()
		showWhileViewing := false

		created, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			BrowserWorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: factoryID,
				EventTypes:  []string{models.NotificationTypeWorkOrderCommentOwned},
			}},
			BrowserEventTypes:       []string{models.NotificationTypeWorkOrderAssigned},
			BrowserShowWhileViewing: &showWhileViewing,
		})
		require.NoError(t, err)
		assert.Equal(t, models.NotificationWorkspaceScopeNone, created.WorkspaceScope)
		assert.Equal(t, models.NotificationWorkspaceScopeFiltered, created.BrowserWorkspaceScope)
		assert.False(t, created.BrowserShowWhileViewing)
		require.Len(t, created.BrowserWorkspaceFilters.Data(), 1)
		assert.Equal(t, factoryID, created.BrowserWorkspaceFilters.Data()[0].WorkspaceID)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderCommentOwned}, created.BrowserWorkspaceFilters.Data()[0].EventTypes)

		updated, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
			BrowserEventTypes:     []string{models.NotificationTypeWorkOrderAssigned},
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.Equal(t, models.NotificationWorkspaceScopeAll, updated.BrowserWorkspaceScope)
		assert.Equal(t, []string{models.NotificationTypeWorkOrderAssigned}, updated.BrowserEventTypes.Data())
		assert.Empty(t, updated.BrowserWorkspaceFilters.Data())
		assert.True(t, updated.BrowserShowWhileViewing)
	})

	t.Run("upsert rejects invalid workspace scope", func(t *testing.T) {
		_, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, r.User, models.UserNotificationSettingsParams{
			WorkspaceScope: "everywhere",
		})
		assert.ErrorIs(t, err, models.ErrNotificationWorkspaceScopeInvalid)
	})

	t.Run("batch find returns only users with a row", func(t *testing.T) {
		withRow := support.CreateUser(t, r, r.Organization.ID).ID
		withoutRow := support.CreateUser(t, r, r.Organization.ID).ID

		_, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, withRow, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		require.NoError(t, err)

		found, err := models.FindUserNotificationSettingsForUsers(db, r.Organization.ID, []uuid.UUID{withRow, withoutRow})
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Contains(t, found, withRow)
	})
}

func Test__UserNotificationSettings__Notifies(t *testing.T) {
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	t.Run("defaults notify every type in every workspace", func(t *testing.T) {
		settings := models.DefaultUserNotificationSettings()
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.True(t, settings.Notifies(otherWorkspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderStatusNoteOwned))
	})

	t.Run("none scope blocks every type", func(t *testing.T) {
		settings := models.UserNotificationSettings{WorkspaceScope: models.NotificationWorkspaceScopeNone}
		assert.False(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderAssigned))
	})

	t.Run("all scope keeps a missing type on", func(t *testing.T) {
		settings := models.UserNotificationSettings{WorkspaceScope: models.NotificationWorkspaceScopeAll}
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderCommentOwned))
	})

	t.Run("all scope with a type list honors the list", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     datatypes.NewJSONType([]string{models.NotificationTypeWorkOrderAssigned}),
		}
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.False(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.True(t, settings.Notifies(otherWorkspaceID, models.NotificationTypeWorkOrderAssigned))
	})

	t.Run("all scope with a type list can opt out of status notes alone", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     datatypes.NewJSONType([]string{models.NotificationTypeWorkOrderCommentOwned}),
		}
		assert.False(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderStatusNoteOwned))
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderCommentOwned))
	})

	t.Run("filtered scope requires the workspace and the type", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: datatypes.NewJSONType([]models.NotificationWorkspaceFilter{{
				WorkspaceID: workspaceID.String(),
				EventTypes:  []string{models.NotificationTypeWorkOrderAssigned},
			}}),
		}
		assert.True(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.False(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.False(t, settings.Notifies(otherWorkspaceID, models.NotificationTypeWorkOrderAssigned))
	})
}

func Test__UserNotificationSettings__NotifiesChannel(t *testing.T) {
	workspaceID := uuid.New()
	otherWorkspaceID := uuid.New()

	t.Run("defaults keep the browser channel off", func(t *testing.T) {
		settings := models.DefaultUserNotificationSettings()
		assert.True(t, settings.NotifiesChannel(models.NotificationChannelEmail, workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.False(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.True(t, settings.BrowserShowWhileViewing)
	})

	t.Run("empty browser scope stays off", func(t *testing.T) {
		settings := models.UserNotificationSettings{WorkspaceScope: models.NotificationWorkspaceScopeAll}
		assert.False(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderAssigned))
	})

	t.Run("browser all scope honors the type list", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
			BrowserEventTypes:     datatypes.NewJSONType([]string{models.NotificationTypeWorkOrderAssigned}),
		}
		assert.True(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.False(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.False(t, settings.Notifies(workspaceID, models.NotificationTypeWorkOrderAssigned))
	})

	t.Run("browser filtered scope requires the workspace and the type", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			BrowserWorkspaceFilters: datatypes.NewJSONType([]models.NotificationWorkspaceFilter{{
				WorkspaceID: workspaceID.String(),
				EventTypes:  []string{models.NotificationTypeWorkOrderCommentOwned},
			}}),
		}
		assert.True(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.False(t, settings.NotifiesChannel(models.NotificationChannelBrowser, workspaceID, models.NotificationTypeWorkOrderAssigned))
		assert.False(t, settings.NotifiesChannel(models.NotificationChannelBrowser, otherWorkspaceID, models.NotificationTypeWorkOrderCommentOwned))
		assert.True(t, settings.Notifies(otherWorkspaceID, models.NotificationTypeWorkOrderAssigned))
	})
}
