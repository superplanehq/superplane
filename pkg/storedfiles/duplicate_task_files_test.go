package storedfiles_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestDuplicateTaskFilesCopiesBytesAndRewritesRefs(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)
	target, err := factoryModel.CreateWorkOrder(db, "Copy", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "notes.md",
		ContentType:    "text/markdown",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("keep me"))))

	description := "See ![notes](" + blob.FileRef(file.ID) + ") and <a href=\"" + blob.FileRef(file.ID) + "\">notes</a>"
	var rewritten string
	var copiedKeys []string
	err = db.Transaction(func(tx *gorm.DB) error {
		next, keys, copyErr := storedfiles.DuplicateTaskFiles(
			t.Context(),
			tx,
			provider,
			r.Organization.ID,
			factoryModel.ID,
			target.ID,
			r.User,
			description,
		)
		rewritten = next
		copiedKeys = keys
		return copyErr
	})
	require.NoError(t, err)
	require.Len(t, copiedKeys, 1)

	ids := blob.FileIDsInMarkdown(rewritten)
	require.Len(t, ids, 1)
	assert.NotEqual(t, file.ID, ids[0])
	assert.Equal(t, 2, bytes.Count([]byte(rewritten), []byte(ids[0].String())))

	sourceFile, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, source.ID, *sourceFile.WorkOrderID)

	copied, err := models.FindFile(db, ids[0])
	require.NoError(t, err)
	require.NotNil(t, copied.WorkOrderID)
	assert.Equal(t, target.ID, *copied.WorkOrderID)
	assert.Equal(t, copiedKeys[0], copied.StorageKey)
	assert.Equal(t, "keep me", readFileObject(t, provider, copied.StorageKey))
	assert.Equal(t, "keep me", readFileObject(t, provider, sourceFile.StorageKey))
}

func TestDuplicateTaskFilesCleanupRemovesCopiedObjectWhenCreateFails(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)
	target, err := factoryModel.CreateWorkOrder(db, "Copy", "", &r.User, nil, nil)
	require.NoError(t, err)
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "notes.md",
		ContentType:    "text/markdown",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("keep me"))))

	var copiedKeys []string
	err = db.Transaction(func(tx *gorm.DB) error {
		_, keys, copyErr := storedfiles.DuplicateTaskFiles(
			t.Context(),
			tx,
			provider,
			r.Organization.ID,
			factoryModel.ID,
			target.ID,
			r.User,
			"![notes]("+blob.FileRef(file.ID)+")",
		)
		copiedKeys = keys
		if copyErr != nil {
			return copyErr
		}
		return models.ErrFileNotFound
	})
	require.ErrorIs(t, err, models.ErrFileNotFound)
	require.Len(t, copiedKeys, 1)

	require.NoError(t, storedfiles.ApplyBindResult(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		storedfiles.BindResult{CopiedKeys: copiedKeys},
		err,
	))
	_, headErr := provider.Head(t.Context(), copiedKeys[0])
	assert.ErrorIs(t, headErr, blob.ErrNotFound)
	assert.Equal(t, "keep me", readFileObject(t, provider, file.StorageKey))
}

func readFileObject(t *testing.T, provider blob.Provider, key string) string {
	t.Helper()
	reader, err := provider.Get(t.Context(), key)
	require.NoError(t, err)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(body)
}
