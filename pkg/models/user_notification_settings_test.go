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
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		require.NoError(t, err)
		assert.True(t, created.Enabled)
		assert.Equal(t, models.NotificationWorkspaceScopeAll, created.WorkspaceScope)

		factoryID := uuid.New().String()
		updated, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, models.UserNotificationSettingsParams{
			Enabled:        false,
			WorkspaceScope: models.NotificationWorkspaceScopeSelected,
			FactoryIDs:     []string{factoryID},
			Types:          map[string]bool{models.NotificationTypeWorkOrderAssigned: false},
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)
		assert.False(t, updated.Enabled)
		assert.Equal(t, models.NotificationWorkspaceScopeSelected, updated.WorkspaceScope)
		assert.Equal(t, []string{factoryID}, []string(updated.FactoryIDs))
		assert.False(t, updated.Types.Data()[models.NotificationTypeWorkOrderAssigned])
	})

	t.Run("upsert rejects invalid workspace scope", func(t *testing.T) {
		_, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, r.User, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: "everywhere",
		})
		assert.ErrorIs(t, err, models.ErrNotificationWorkspaceScopeInvalid)
	})

	t.Run("batch find returns only users with a row", func(t *testing.T) {
		withRow := support.CreateUser(t, r, r.Organization.ID).ID
		withoutRow := support.CreateUser(t, r, r.Organization.ID).ID

		_, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, withRow, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		require.NoError(t, err)

		found, err := models.FindUserNotificationSettingsForUsers(db, r.Organization.ID, []uuid.UUID{withRow, withoutRow})
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Contains(t, found, withRow)
	})
}

func Test__UserNotificationSettings__NotifiesType(t *testing.T) {
	t.Run("defaults enable every type in every workspace", func(t *testing.T) {
		settings := models.DefaultUserNotificationSettings()
		assert.True(t, settings.Enabled)
		assert.True(t, settings.NotifiesType(models.NotificationTypeWorkOrderAssigned))
		assert.True(t, settings.AppliesToFactory(uuid.New()))
	})

	t.Run("master switch off blocks every type", func(t *testing.T) {
		settings := models.UserNotificationSettings{Enabled: false}
		assert.False(t, settings.NotifiesType(models.NotificationTypeWorkOrderAssigned))
	})

	t.Run("missing type key defaults to on", func(t *testing.T) {
		settings := models.UserNotificationSettings{Enabled: true}
		assert.True(t, settings.NotifiesType(models.NotificationTypeWorkOrderCommentOwned))
	})

	t.Run("explicit toggle wins", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			Enabled: true,
			Types: datatypes.NewJSONType(map[string]bool{
				models.NotificationTypeWorkOrderCommentOwned: false,
			}),
		}
		assert.False(t, settings.NotifiesType(models.NotificationTypeWorkOrderCommentOwned))
		assert.True(t, settings.NotifiesType(models.NotificationTypeWorkOrderStatusOwned))
	})
}

func Test__UserNotificationSettings__AppliesToFactory(t *testing.T) {
	selected := uuid.New()
	other := uuid.New()

	t.Run("all scope covers every factory", func(t *testing.T) {
		settings := models.UserNotificationSettings{WorkspaceScope: models.NotificationWorkspaceScopeAll}
		assert.True(t, settings.AppliesToFactory(other))
	})

	t.Run("selected scope only covers listed factories", func(t *testing.T) {
		settings := models.UserNotificationSettings{
			WorkspaceScope: models.NotificationWorkspaceScopeSelected,
			FactoryIDs:     datatypes.NewJSONSlice([]string{selected.String()}),
		}
		assert.True(t, settings.AppliesToFactory(selected))
		assert.False(t, settings.AppliesToFactory(other))
	})
}
