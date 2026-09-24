package files

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/files"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func TestCreateAndListFactoryFile(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	created, err := CreateFactoryFile(ctx, r.Organization.ID.String(), &pb.CreateFactoryFileRequest{
		FactoryId:   factoryModel.ID.String(),
		Filename:    "shot.png",
		ContentType: "image/png",
	})
	require.NoError(t, err)
	require.NotNil(t, created.File)
	assert.Equal(t, "pending", created.File.State)
	assert.Contains(t, created.File.UploadUrl, "/api/v1/files/")

	fileID, err := uuid.Parse(created.File.Id)
	require.NoError(t, err)
	file, err := models.FindFile(database.Conn(), fileID)
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

	listed, err := ListFactoryFiles(ctx, r.Organization.ID.String(), &pb.ListFactoryFilesRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, listed.Files, 1)
	assert.Equal(t, "ready", listed.Files[0].State)
	assert.NotEmpty(t, listed.Files[0].DownloadUrl)
}
