package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CanvasMemoryContext(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	ctx := NewCanvasMemoryContext(database.Conn(), canvas.ID)
	require.NotNil(t, ctx)

	apiValues := map[string]any{"service": "api", "status": "queued", "env": "prod"}
	apiRecord, err := ctx.AddRecord(" services ", apiValues)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, apiRecord.ID)
	assert.Equal(t, apiValues, apiRecord.Values)

	workerValues := map[string]any{"service": "worker", "status": "queued", "env": "prod"}
	require.NoError(t, ctx.Add("services", workerValues))

	found, err := ctx.Find(" services ", map[string]any{"env": "prod"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []any{apiValues, workerValues}, found)

	first, err := ctx.FindFirst("services", map[string]any{"service": "api"})
	require.NoError(t, err)
	assert.Equal(t, apiValues, first)

	updatedRecords, err := ctx.UpdateRecords("services", map[string]any{"service": "api"}, map[string]any{"status": "running"})
	require.NoError(t, err)
	require.Len(t, updatedRecords, 1)
	assert.Equal(t, apiRecord.ID, updatedRecords[0].ID)
	assert.Equal(t, map[string]any{"service": "api", "status": "running", "env": "prod"}, updatedRecords[0].Values)

	updatedValues, err := ctx.Update("services", map[string]any{"service": "worker"}, map[string]any{"status": "running"})
	require.NoError(t, err)
	assert.Equal(t, []any{
		map[string]any{"service": "worker", "status": "running", "env": "prod"},
	}, updatedValues)

	namespaceRecords, err := ctx.UpdateNamespaceRecords("services", map[string]any{"batch": "done"})
	require.NoError(t, err)
	require.Len(t, namespaceRecords, 2)
	for _, record := range namespaceRecords {
		values, ok := record.Values.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "done", values["batch"])
	}

	namespaceValues, err := ctx.UpdateNamespace("services", map[string]any{"generation": "2"})
	require.NoError(t, err)
	require.Len(t, namespaceValues, 2)
	for _, value := range namespaceValues {
		values, ok := value.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "2", values["generation"])
	}

	deleted, err := ctx.Delete("services", map[string]any{"service": "api"})
	require.NoError(t, err)
	assert.Equal(t, []any{
		map[string]any{"service": "api", "status": "running", "env": "prod", "batch": "done", "generation": "2"},
	}, deleted)

	missing, err := ctx.FindFirst("services", map[string]any{"service": "api"})
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func Test__CanvasMemoryContext__RejectsBlankNamespace(t *testing.T) {
	ctx := NewCanvasMemoryContext(database.Conn(), uuid.New())

	err := ctx.Add(" ", map[string]any{"service": "api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.AddRecord(" ", map[string]any{"service": "api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.Find(" ", map[string]any{"service": "api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.FindFirst(" ", map[string]any{"service": "api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.Delete(" ", map[string]any{"service": "api"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.Update(" ", map[string]any{"service": "api"}, map[string]any{"status": "running"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.UpdateRecords(" ", map[string]any{"service": "api"}, map[string]any{"status": "running"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.UpdateNamespace(" ", map[string]any{"status": "running"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")

	_, err = ctx.UpdateNamespaceRecords(" ", map[string]any{"status": "running"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace is required")
}

func Test__CanvasMemoryContext__RecordConversionHelpers(t *testing.T) {
	id := uuid.New()
	record := models.CanvasMemory{
		ID:     id,
		Values: datatypes.NewJSONType[any](map[string]any{"service": "api"}),
	}

	converted := canvasMemoryRecord(record)
	assert.Equal(t, id, converted.ID)
	assert.Equal(t, map[string]any{"service": "api"}, converted.Values)

	records := canvasMemoryRecords([]models.CanvasMemory{record})
	assert.Equal(t, []any{map[string]any{"service": "api"}}, canvasMemoryRecordValues(records))
}
