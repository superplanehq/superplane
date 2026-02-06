package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestCanvasData_SetGetAndHistory(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	canvasID := uuid.New()
	// Create a workflow (canvas) so FK is satisfied
	wf := &Canvas{
		ID:             canvasID,
		OrganizationID: uuid.New(),
		Name:           "Test Canvas",
		Description:    "Test",
	}
	require.NoError(t, database.Conn().Create(wf).Error)

	key := "app/test/version"

	t.Run("SetCanvasData creates entry", func(t *testing.T) {
		rec, err := SetCanvasData(canvasID, key, "v1")
		require.NoError(t, err)
		require.NotNil(t, rec)
		require.Equal(t, key, rec.Key)
		require.Equal(t, "v1", rec.Value)
		require.NotNil(t, rec.CreatedAt)
	})

	t.Run("GetCanvasData current returns latest", func(t *testing.T) {
		rec, err := GetCanvasData(canvasID, key, 0)
		require.NoError(t, err)
		require.NotNil(t, rec)
		require.Equal(t, "v1", rec.Value)
	})

	t.Run("SetCanvasData again adds history", func(t *testing.T) {
		_, err := SetCanvasData(canvasID, key, "v2")
		require.NoError(t, err)
	})

	t.Run("GetCanvasData current returns v2", func(t *testing.T) {
		rec, err := GetCanvasData(canvasID, key, 0)
		require.NoError(t, err)
		require.NotNil(t, rec)
		require.Equal(t, "v2", rec.Value)
	})

	t.Run("GetCanvasData previous returns v1", func(t *testing.T) {
		rec, err := GetCanvasData(canvasID, key, 1)
		require.NoError(t, err)
		require.NotNil(t, rec)
		require.Equal(t, "v1", rec.Value)
	})

	t.Run("GetCanvasData for missing key returns nil", func(t *testing.T) {
		rec, err := GetCanvasData(canvasID, "nonexistent", 0)
		require.NoError(t, err)
		require.Nil(t, rec)
	})

	t.Run("ListCanvasDataHistory returns newest first", func(t *testing.T) {
		recs, err := ListCanvasDataHistory(canvasID, key, 10)
		require.NoError(t, err)
		require.Len(t, recs, 2)
		require.Equal(t, "v2", recs[0].Value)
		require.Equal(t, "v1", recs[1].Value)
	})
}

func TestCanvasData_InTransaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	canvasID := uuid.New()
	wf := &Canvas{
		ID:             canvasID,
		OrganizationID: uuid.New(),
		Name:           "Test Canvas",
		Description:    "Test",
	}
	require.NoError(t, database.Conn().Create(wf).Error)

	t.Run("SetCanvasDataInTransaction and GetCanvasDataInTransaction", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		rec, err := SetCanvasDataInTransaction(tx, canvasID, "tx/key", "value")
		require.NoError(t, err)
		require.NotNil(t, rec)
		require.Equal(t, "value", rec.Value)

		got, err := GetCanvasDataInTransaction(tx, canvasID, "tx/key", 0)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "value", got.Value)
	})

	t.Run("rollback leaves no data", func(t *testing.T) {
		func() {
			tx := database.Conn().Begin()
			defer tx.Rollback()
			_, err := SetCanvasDataInTransaction(tx, canvasID, "rollback/key", "x")
			require.NoError(t, err)
		}()
		rec, err := GetCanvasData(canvasID, "rollback/key", 0)
		require.NoError(t, err)
		require.Nil(t, rec)
	})
}

func TestCanvasData_GetCanvasData_offsetBeyondHistory(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	canvasID := uuid.New()
	wf := &Canvas{
		ID:             canvasID,
		OrganizationID: uuid.New(),
		Name:           "Test Canvas",
		Description:    "Test",
	}
	require.NoError(t, database.Conn().Create(wf).Error)

	_, err := SetCanvasData(canvasID, "only-one", "v1")
	require.NoError(t, err)

	rec, err := GetCanvasData(canvasID, "only-one", 1)
	require.NoError(t, err)
	require.Nil(t, rec)

	rec, err = GetCanvasData(canvasID, "only-one", 0)
	require.NoError(t, err)
	require.NotNil(t, rec)
	require.Equal(t, "v1", rec.Value)
}

// Ensure we don't break when record not found
func TestCanvasData_GetCanvasDataInTransaction_notFound(t *testing.T) {
	tx := database.Conn().Begin()
	defer tx.Rollback()

	rec, err := GetCanvasDataInTransaction(tx, uuid.New(), "nokey", 0)
	require.NoError(t, err)
	require.Nil(t, rec)
}
