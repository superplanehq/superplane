package contexts

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func TestWorkOrderCreatedPayloadRewritesFileRefs(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bridge.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("png-bytes"))))

	description := "See ![bridge](" + blob.FileRef(file.ID) + ")"
	order, err := factoryModel.CreateWorkOrder(db, "Score this", description, &r.User, nil, nil)
	require.NoError(t, err)

	payload := workOrderCreatedPayload(db, order)
	workOrder, ok := payload["workOrder"].(map[string]any)
	require.True(t, ok)

	rewritten, ok := workOrder["description"].(string)
	require.True(t, ok)
	assert.NotContains(t, rewritten, blob.FileRef(file.ID))
	assert.Contains(t, rewritten, "/api/v1/public/files/"+file.ID.String())

	files, ok := workOrder["files"].([]any)
	require.True(t, ok)
	require.Len(t, files, 1)
	item, ok := files[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, file.ID.String(), item["id"])
	assert.Equal(t, "bridge.png", item["filename"])
	url, ok := item["url"].(string)
	require.True(t, ok)
	assert.Contains(t, url, "/api/v1/public/files/"+file.ID.String())
	assert.Equal(t, description, order.Description)
}

func TestWorkOrderCreatedPayloadKeepsRawDescriptionWhenMintFails(t *testing.T) {
	r := support.Setup(t)
	blob.SetCurrent(nil)

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	fileID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	description := "See ![bridge](sp-file://" + fileID + ")"
	order, err := factoryModel.CreateWorkOrder(db, "Score this", description, &r.User, nil, nil)
	require.NoError(t, err)

	payload := workOrderCreatedPayload(db, order)
	workOrder, ok := payload["workOrder"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, description, workOrder["description"])
	assert.Equal(t, []any{}, workOrder["files"])
}
