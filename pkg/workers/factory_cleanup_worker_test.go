package workers

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryCleanupWorker_DeletesFileObjectsBeforeRows(t *testing.T) {
	r := support.Setup(t)
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(database.Conn(), "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	var files []*models.File
	for i := 0; i < 2; i++ {
		file, createErr := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
			Scope:          blob.ScopeTask,
			OrganizationID: r.Organization.ID,
			FactoryID:      factory.ID,
			WorkOrderID:    order.ID,
			Filename:       "shot.png",
			ContentType:    "image/png",
			CreatedByID:    r.User,
		})
		require.NoError(t, createErr)
		require.NoError(t, storedfiles.CompleteUpload(
			t.Context(),
			database.Conn(),
			store,
			file,
			bytes.NewReader([]byte("png-bytes")),
		))
		files = append(files, file)
	}

	require.NoError(t, factory.SoftDelete(database.Conn()))
	deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).
		Where("id = ?", factory.ID).
		Update("deleted_at", deletedAtOutsideGracePeriod).
		Error)
	factory.DeletedAt.Time = deletedAtOutsideGracePeriod
	factory.DeletedAt.Valid = true

	worker := NewFactoryCleanupWorker()
	worker.maxResourcesPerTick = 1
	require.NoError(t, worker.LockAndProcessFactory(*factory))

	var remaining int64
	require.NoError(t, database.Conn().Model(&models.File{}).Where("factory_id = ?", factory.ID).Count(&remaining).Error)
	assert.Equal(t, int64(1), remaining)

	var factoryCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
	assert.Equal(t, int64(1), factoryCount)

	worker.maxResourcesPerTick = 500
	require.NoError(t, worker.LockAndProcessFactory(*factory))

	require.NoError(t, database.Conn().Model(&models.File{}).Where("factory_id = ?", factory.ID).Count(&remaining).Error)
	assert.Equal(t, int64(0), remaining)
	require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
	assert.Equal(t, int64(0), factoryCount)

	for _, file := range files {
		_, headErr := store.Head(t.Context(), file.StorageKey)
		assert.ErrorIs(t, headErr, blob.ErrNotFound)
	}
}

func Test__FactoryCleanupWorker_GracePeriod(t *testing.T) {
	r := support.Setup(t)

	t.Run("skips cleanup while factory is still within grace period", func(t *testing.T) {
		worker := NewFactoryCleanupWorker()
		factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		require.NoError(t, factory.SoftDelete(database.Conn()))

		deletedAtWithinGracePeriod := time.Now().AddDate(0, 0, -29)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).
			Where("id = ?", factory.ID).
			Update("deleted_at", deletedAtWithinGracePeriod).
			Error)

		factory.DeletedAt.Time = deletedAtWithinGracePeriod
		factory.DeletedAt.Valid = true
		require.NoError(t, worker.LockAndProcessFactory(*factory))

		var factoryCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
		assert.Equal(t, int64(1), factoryCount)
	})

	t.Run("hard deletes factory after grace when no apps remain", func(t *testing.T) {
		worker := NewFactoryCleanupWorker()
		factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		order, err := factory.CreateWorkOrder(database.Conn(), "Order", "", &r.User, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, order)

		require.NoError(t, factory.SoftDelete(database.Conn()))
		deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).
			Where("id = ?", factory.ID).
			Update("deleted_at", deletedAtOutsideGracePeriod).
			Error)

		factory.DeletedAt.Time = deletedAtOutsideGracePeriod
		factory.DeletedAt.Valid = true
		require.NoError(t, worker.LockAndProcessFactory(*factory))

		var factoryCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
		assert.Equal(t, int64(0), factoryCount)
	})

	t.Run("waits while factory apps still exist", func(t *testing.T) {
		worker := NewFactoryCleanupWorker()
		factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factory.ID).Error)

		require.NoError(t, factory.SoftDelete(database.Conn()))
		deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).
			Where("id = ?", factory.ID).
			Update("deleted_at", deletedAtOutsideGracePeriod).
			Error)

		factory.DeletedAt.Time = deletedAtOutsideGracePeriod
		factory.DeletedAt.Valid = true
		require.NoError(t, worker.LockAndProcessFactory(*factory))

		var factoryCount int64
		require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
		assert.Equal(t, int64(1), factoryCount)

		deletedCanvas, err := models.FindUnscopedCanvas(canvas.ID)
		require.NoError(t, err)
		assert.True(t, deletedCanvas.DeletedAt.Valid)
	})
}

func Test__OrganizationCleanupWorker_WaitsForFactories(t *testing.T) {
	r := support.Setup(t)

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = factory.CreateWorkOrder(database.Conn(), "Order", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)

	worker := NewOrganizationCleanupWorker(r.GitProvider)
	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))
	deletedAtOutsideGracePeriod := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).
		Where("id = ?", r.Organization.ID).
		Update("deleted_at", deletedAtOutsideGracePeriod).
		Error)

	deletedOrganizations, err := models.ListDeletedOrganizations()
	require.NoError(t, err)
	require.Len(t, deletedOrganizations, 1)

	require.NoError(t, worker.LockAndProcessOrganization(deletedOrganizations[0]))

	var organizationCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Count(&organizationCount).Error)
	assert.Equal(t, int64(1), organizationCount)

	var factoryCount int64
	require.NoError(t, database.Conn().Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
	assert.Equal(t, int64(1), factoryCount)

	deletedFactory, err := models.FindFactory(database.Conn().Unscoped(), r.Organization.ID, factory.ID)
	require.NoError(t, err)
	assert.True(t, deletedFactory.DeletedAt.Valid)
}
