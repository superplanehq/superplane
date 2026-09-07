package models

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestCreatePendingFileRejectsDisallowedContentType(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "file-type")

	_, err := CreatePendingFile(database.Conn(), CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: org.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "clip.mp4",
		ContentType:    "video/mp4",
		CreatedByID:    userID,
	})
	assert.ErrorIs(t, err, ErrFileContentType)
}

func TestCreatePendingFileStoresWorkspaceScope(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "file-create")

	file, err := CreatePendingFile(database.Conn(), CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: org.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    userID,
	})
	require.NoError(t, err)
	assert.Equal(t, FileStatePending, file.State)
	assert.Equal(t, blob.ScopeWorkspace, file.Scope)
	assert.Equal(t, "bug.png", file.Filename)
	assert.Nil(t, file.WorkOrderID)
	assert.Contains(t, file.StorageKey, "/workspaces/"+factoryModel.ID.String()+"/")
}

func TestReparentToTaskUpdatesScopeAndKey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "file-reparent")
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "With file", "", &userID, nil, nil)
	require.NoError(t, err)

	file, err := CreatePendingFile(database.Conn(), CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: org.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "notes.pdf",
		ContentType:    "application/pdf",
		CreatedByID:    userID,
	})
	require.NoError(t, err)
	require.NoError(t, file.MarkReady(database.Conn(), 12, "abc"))

	nextKey, err := blob.ObjectKey(file.InstallationID, blob.ScopeTask, org.ID, factoryModel.ID, order.ID, file.ID)
	require.NoError(t, err)
	require.NoError(t, file.ReparentToTask(database.Conn(), order.ID, nextKey))

	loaded, err := FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, loaded.Scope)
	assert.Equal(t, order.ID, *loaded.WorkOrderID)
	assert.Equal(t, nextKey, loaded.StorageKey)
}

func TestEnsureFileQuotaRejectsTooManyTaskFiles(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "file-quota")
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Quota", "", &userID, nil, nil)
	require.NoError(t, err)

	for i := 0; i < MaxFilesPerWorkOrder; i++ {
		file, createErr := CreatePendingFile(database.Conn(), CreateFileParams{
			Scope:          blob.ScopeTask,
			OrganizationID: org.ID,
			FactoryID:      factoryModel.ID,
			WorkOrderID:    order.ID,
			Filename:       "shot.png",
			ContentType:    "image/png",
			CreatedByID:    userID,
		})
		require.NoError(t, createErr)
		require.NoError(t, file.MarkReady(database.Conn(), 1, "x"))
	}

	_, err = CreatePendingFile(database.Conn(), CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: org.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		Filename:       "extra.png",
		ContentType:    "image/png",
		CreatedByID:    userID,
	})
	assert.ErrorIs(t, err, ErrFileQuotaExceeded)
}

func TestFilesystemRoundTripUsedByFileCatalog(t *testing.T) {
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Put(t.Context(), "k", bytes.NewReader([]byte("hi")), blob.PutOptions{ContentType: "text/plain"}))
	info, err := store.Head(t.Context(), "k")
	require.NoError(t, err)
	assert.Equal(t, int64(2), info.Size)
}
