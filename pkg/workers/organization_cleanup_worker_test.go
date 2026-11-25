package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CanvasCleanupWorker_GracePeriod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Run("skips cleanup while canvas is still within grace period", func(t *testing.T) {
		worker := NewCanvasCleanupWorker()
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		require.NoError(t, canvas.SoftDelete())
		deletedAtWithinGracePeriod := time.Now().AddDate(0, 0, -29)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", deletedAtWithinGracePeriod).Error)

		deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)

		require.NoError(t, worker.LockAndProcessCanvas(*deletedCanvas))

		var canvasCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
		assert.Equal(t, int64(1), canvasCount)
	})

	t.Run("cleans up canvas after grace period expires", func(t *testing.T) {
		worker := NewCanvasCleanupWorker()
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		require.NoError(t, canvas.SoftDelete())
		deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("deleted_at", deletedAtOutsideGracePeriod).Error)

		deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)

		require.NoError(t, worker.LockAndProcessCanvas(*deletedCanvas))

		var canvasCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
		assert.Equal(t, int64(0), canvasCount)
	})
}

func Test__OrganizationCleanupWorker_GracePeriod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Run("skips cleanup while organization is still within grace period", func(t *testing.T) {
		worker := NewOrganizationCleanupWorker()
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))
		deletedAtWithinGracePeriod := time.Now().AddDate(0, 0, -29)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("deleted_at", deletedAtWithinGracePeriod).Error)

		deletedOrganizations, err := models.ListDeletedOrganizations()
		require.NoError(t, err)
		require.Len(t, deletedOrganizations, 1)

		require.NoError(t, worker.LockAndProcessOrganization(deletedOrganizations[0]))

		var organizationCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&organizationCount).Error)
		assert.Equal(t, int64(1), organizationCount)

		var canvasCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
		assert.Equal(t, int64(1), canvasCount)
	})

	t.Run("cleans up organization after grace period expires", func(t *testing.T) {
		r2 := support.Setup(t)
		defer r2.Close()

		canvasWorker := NewCanvasCleanupWorker()
		worker := NewOrganizationCleanupWorker()
		canvas, _ := support.CreateCanvas(t, r2.Organization.ID, r2.User, []models.CanvasNode{}, []models.Edge{})

		require.NoError(t, models.SoftDeleteOrganization(r2.Organization.ID.String()))
		deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r2.Organization.ID).Update("deleted_at", deletedAtOutsideGracePeriod).Error)

		deletedCanvases, err := models.ListDeletedCanvases(database.Conn())
		require.NoError(t, err)
		require.Len(t, deletedCanvases, 1)
		require.Equal(t, canvas.ID, deletedCanvases[0].ID)
		require.NoError(t, canvasWorker.LockAndProcessCanvas(deletedCanvases[0]))

		deletedOrganizations, err := models.ListDeletedOrganizations()
		require.NoError(t, err)
		require.Len(t, deletedOrganizations, 1)

		require.NoError(t, worker.LockAndProcessOrganization(deletedOrganizations[0]))

		var organizationCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r2.Organization.ID).Count(&organizationCount).Error)
		assert.Equal(t, int64(0), organizationCount)

		var canvasCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
		assert.Equal(t, int64(0), canvasCount)

		var userCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.User{}).Where("organization_id = ?", r2.Organization.ID).Count(&userCount).Error)
		assert.Equal(t, int64(0), userCount)

	})

	t.Run("cleans up an organization that has SSO identities", func(t *testing.T) {
		r3 := support.Setup(t)
		defer r3.Close()

		canvasWorker := NewCanvasCleanupWorker()
		worker := NewOrganizationCleanupWorker()
		canvas, _ := support.CreateCanvas(t, r3.Organization.ID, r3.User, []models.CanvasNode{}, []models.Edge{})
		now := time.Now()
		connection := &models.SSOConnection{
			ID:                   uuid.New(),
			OrganizationID:       r3.Organization.ID,
			Protocol:             models.SSOProtocolOIDC,
			Preset:               models.SSOPresetOkta,
			DisplayName:          "Okta",
			Status:               models.SSOStatusActive,
			Enforcement:          models.SSOEnforcementOptional,
			AllowOwnerBypass:     true,
			JITEnabled:           true,
			DefaultRole:          models.RoleOrgOperator,
			MaxSessionAgeSeconds: models.SSODefaultSessionAge,
			Config:               datatypes.JSON(`{}`),
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		require.NoError(t, database.Conn().Create(connection).Error)
		require.NoError(t, database.Conn().Create(&models.OrganizationDomain{
			OrganizationID: r3.Organization.ID, Domain: "cleanup.example", VerificationToken: "token",
			CreatedAt: now, UpdatedAt: now,
		}).Error)
		require.NoError(t, database.Conn().Create(&models.SCIMToken{
			ConnectionID: connection.ID, TokenHash: uuid.NewString(), Description: "Okta", CreatedAt: now,
		}).Error)
		subject := "subject-1"
		userID := r3.User
		require.NoError(t, database.Conn().Create(&models.SSOIdentity{
			ConnectionID: connection.ID, OrganizationID: r3.Organization.ID, AccountID: r3.Account.ID,
			UserID: &userID, Subject: &subject, Email: "ada@example.com", Active: true,
			ProvisionedBy: models.SSOProvisionedByJIT, CreatedAt: now, UpdatedAt: now,
		}).Error)

		require.NoError(t, models.SoftDeleteOrganization(r3.Organization.ID.String()))
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r3.Organization.ID).Update("deleted_at", time.Now().AddDate(0, 0, -31)).Error)
		deletedCanvases, err := models.ListDeletedCanvases(database.Conn())
		require.NoError(t, err)
		require.Len(t, deletedCanvases, 1)
		require.NoError(t, canvasWorker.LockAndProcessCanvas(deletedCanvases[0]))
		deletedOrganizations, err := models.ListDeletedOrganizations()
		require.NoError(t, err)
		require.Len(t, deletedOrganizations, 1)
		require.NoError(t, worker.LockAndProcessOrganization(deletedOrganizations[0]))

		var organizationCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r3.Organization.ID).Count(&organizationCount).Error)
		assert.Equal(t, int64(0), organizationCount)

		var canvasCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount).Error)
		assert.Equal(t, int64(0), canvasCount)
	})
}
