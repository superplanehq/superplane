package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__OrganizationCleanupWorker_SkipsOrganizationWithinGracePeriod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := &OrganizationCleanupWorker{
		logger:      newTestLogger("OrganizationCleanupWorker"),
		gracePeriod: 30 * 24 * time.Hour,
	}

	err := models.SoftDeleteOrganization(r.Organization.ID.String())
	require.NoError(t, err)

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)
	require.True(t, org.DeletedAt.Valid)

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	var count int64
	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&count)
	assert.Equal(t, int64(1), count, "Organization should still exist within grace period")
}

func Test__OrganizationCleanupWorker_HardDeletesAfterGracePeriod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := &OrganizationCleanupWorker{
		logger:      newTestLogger("OrganizationCleanupWorker"),
		gracePeriod: 0,
	}

	err := models.SoftDeleteOrganization(r.Organization.ID.String())
	require.NoError(t, err)

	// Soft-delete users so they don't block hard-delete
	err = models.SoftDeleteOrganizationUsersInTransaction(database.Conn(), r.Organization.ID.String())
	require.NoError(t, err)

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	var count int64
	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&count)
	assert.Equal(t, int64(0), count, "Organization should be hard-deleted after grace period")
}

func Test__OrganizationCleanupWorker_WaitsForChildResources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := &OrganizationCleanupWorker{
		logger:      newTestLogger("OrganizationCleanupWorker"),
		gracePeriod: 0,
	}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	err := models.SoftDeleteOrganization(r.Organization.ID.String())
	require.NoError(t, err)

	err = canvas.SoftDelete()
	require.NoError(t, err)

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	var orgCount int64
	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&orgCount)
	assert.Equal(t, int64(1), orgCount, "Organization should not be hard-deleted while canvases remain")

	canvasWorker := NewCanvasCleanupWorker()
	deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		err = canvasWorker.LockAndProcessCanvas(*deletedCanvas)
		require.NoError(t, err)

		var canvasCount int64
		database.Conn().Unscoped().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Count(&canvasCount)
		if canvasCount == 0 {
			break
		}
	}

	// Soft-delete users so they don't block hard-delete
	err = models.SoftDeleteOrganizationUsersInTransaction(database.Conn(), r.Organization.ID.String())
	require.NoError(t, err)

	// Hard-delete users since the org cleanup worker checks for remaining users
	database.Conn().Unscoped().Where("organization_id = ?", r.Organization.ID).Delete(&models.User{})

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&orgCount)
	assert.Equal(t, int64(0), orgCount, "Organization should be hard-deleted after all child resources are cleaned")
}

func Test__OrganizationCleanupWorker_IgnoresNonDeletedOrganization(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := &OrganizationCleanupWorker{
		logger:      newTestLogger("OrganizationCleanupWorker"),
		gracePeriod: 0,
	}

	err := worker.LockAndProcessOrganization(*r.Organization)
	require.NoError(t, err)

	var count int64
	database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&count)
	assert.Equal(t, int64(1), count, "Non-deleted organization should not be affected")
}

func Test__OrganizationCleanupWorker_WaitsForIntegrations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := &OrganizationCleanupWorker{
		logger:      newTestLogger("OrganizationCleanupWorker"),
		gracePeriod: 0,
	}

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		"test-integration",
		map[string]any{"key": "value"},
	)
	require.NoError(t, err)

	err = models.SoftDeleteOrganization(r.Organization.ID.String())
	require.NoError(t, err)

	err = integration.SoftDelete()
	require.NoError(t, err)

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	var orgCount int64
	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&orgCount)
	assert.Equal(t, int64(1), orgCount, "Organization should not be hard-deleted while integrations remain")

	database.Conn().Unscoped().Where("id = ?", integration.ID).Delete(&models.Integration{})

	// Soft-delete and hard-delete users
	models.SoftDeleteOrganizationUsersInTransaction(database.Conn(), r.Organization.ID.String())
	database.Conn().Unscoped().Where("organization_id = ?", r.Organization.ID).Delete(&models.User{})

	err = worker.LockAndProcessOrganization(org)
	require.NoError(t, err)

	database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&orgCount)
	assert.Equal(t, int64(0), orgCount, "Organization should be hard-deleted after integrations are cleaned")
}

func Test__OrganizationCleanupWorker_HandlesConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	err := models.SoftDeleteOrganization(r.Organization.ID.String())
	require.NoError(t, err)

	models.SoftDeleteOrganizationUsersInTransaction(database.Conn(), r.Organization.ID.String())
	database.Conn().Unscoped().Where("organization_id = ?", r.Organization.ID).Delete(&models.User{})

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)

	results := make(chan error, 2)

	go func() {
		worker := &OrganizationCleanupWorker{
			logger:      newTestLogger("OrganizationCleanupWorker"),
			gracePeriod: 0,
		}
		results <- worker.LockAndProcessOrganization(org)
	}()

	go func() {
		worker := &OrganizationCleanupWorker{
			logger:      newTestLogger("OrganizationCleanupWorker"),
			gracePeriod: 0,
		}
		results <- worker.LockAndProcessOrganization(org)
	}()

	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)
}

func newTestLogger(name string) *log.Entry {
	return log.WithFields(log.Fields{"worker": name})
}
