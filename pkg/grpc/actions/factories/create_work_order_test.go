package factories

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CreateWorkOrder__AssignsTheCreator(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	other := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Ship the refunds line",
		AssigneeIds: []string{other.ID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
}

func Test__CreateWorkOrder__ReparentsWorkspaceFiles(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Login screenshot",
		Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.Equal(t, file.ID.String(), resp.Order.Files[0].Id)
	assert.NotEmpty(t, resp.Order.Files[0].DownloadUrl)
	assert.Contains(t, resp.Order.Description, blob.FileRef(file.ID))
	assert.NotContains(t, resp.Order.Description, "http://")

	reparented, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, reparented.Scope)
}

func Test__CreateWorkOrder__CopiesTaskFilesWithoutMovingTheSource(t *testing.T) {
	r := support.Setup(t)
	store := setupCreateWorkOrderFileStore(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, db, store, file, bytes.NewReader([]byte("png-bytes"))))
	sourceKey := file.StorageKey
	ref := blob.FileRef(file.ID)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Login screenshot",
		Description: "See ![bug](" + ref + ") and again [bug.png](" + ref + ") plus [docs](https://example.com).",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.NotEqual(t, file.ID.String(), resp.Order.Files[0].Id)
	assert.Equal(t, "bug.png", resp.Order.Files[0].Filename)
	assert.Equal(t, "image/png", resp.Order.Files[0].ContentType)
	assert.NotContains(t, resp.Order.Description, ref)
	assert.Contains(t, resp.Order.Description, blob.FileRef(uuid.MustParse(resp.Order.Files[0].Id)))
	assert.Equal(t, 2, bytes.Count([]byte(resp.Order.Description), []byte(resp.Order.Files[0].Id)))
	assert.Contains(t, resp.Order.Description, "[docs](https://example.com)")

	unchanged, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, source.ID, *unchanged.WorkOrderID)
	assert.Equal(t, sourceKey, unchanged.StorageKey)
	assert.Equal(t, "png-bytes", readStoredObject(t, store, sourceKey))

	copied, err := models.FindFile(db, uuid.MustParse(resp.Order.Files[0].Id))
	require.NoError(t, err)
	require.NotNil(t, copied.WorkOrderID)
	assert.Equal(t, resp.Order.GetId(), copied.WorkOrderID.String())
	assert.Equal(t, "png-bytes", readStoredObject(t, store, copied.StorageKey))
	assert.NotEqual(t, sourceKey, copied.StorageKey)
}

func Test__CreateWorkOrder__KeepsSourceFileWhenCopyFails(t *testing.T) {
	r := support.Setup(t)
	store := setupCreateWorkOrderFileStore(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, db, store, file, bytes.NewReader([]byte("png-bytes"))))

	_, err = CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Broken copy",
		Description: "![bug](" + blob.FileRef(file.ID) + ") ![missing](" + blob.FileRef(uuid.New()) + ")",
	})
	require.Error(t, err)

	unchanged, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, source.ID, *unchanged.WorkOrderID)
	assert.Equal(t, "png-bytes", readStoredObject(t, store, unchanged.StorageKey))
}

func setupCreateWorkOrderFileStore(t *testing.T) blob.Provider {
	t.Helper()
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	return store
}

func readStoredObject(t *testing.T, store blob.Provider, key string) string {
	t.Helper()
	reader, err := store.Get(t.Context(), key)
	require.NoError(t, err)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(body)
}
