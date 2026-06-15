package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAgentMemoryStoreScopeIsUniquePerProvider(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	store := &models.AgentMemoryStore{
		OrganizationID:        r.Organization.ID,
		UserID:                r.User,
		CanvasID:              canvas.ID,
		Provider:              "anthropic",
		ProviderMemoryStoreID: "memstore_1",
		Name:                  "memory",
	}
	require.NoError(t, models.CreateAgentMemoryStoreInTransaction(database.Conn(), store))

	duplicate := &models.AgentMemoryStore{
		OrganizationID:        r.Organization.ID,
		UserID:                r.User,
		CanvasID:              canvas.ID,
		Provider:              "anthropic",
		ProviderMemoryStoreID: "memstore_2",
		Name:                  "memory 2",
	}
	require.Error(t, models.CreateAgentMemoryStoreInTransaction(database.Conn(), duplicate))
}

func TestFindAgentMemoryStoreByScope(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	store := &models.AgentMemoryStore{
		OrganizationID:        r.Organization.ID,
		UserID:                r.User,
		CanvasID:              canvas.ID,
		Provider:              "anthropic",
		ProviderMemoryStoreID: "memstore_1",
		Name:                  "memory",
	}
	require.NoError(t, models.CreateAgentMemoryStoreInTransaction(database.Conn(), store))

	found, err := models.FindAgentMemoryStoreByScope(r.Organization.ID, r.User, canvas.ID, "anthropic")
	require.NoError(t, err)
	assert.Equal(t, store.ProviderMemoryStoreID, found.ProviderMemoryStoreID)
}

func TestAgentMemoryStoreCascadesWhenCanvasIsDeleted(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	store := &models.AgentMemoryStore{
		OrganizationID:        r.Organization.ID,
		UserID:                r.User,
		CanvasID:              canvas.ID,
		Provider:              "anthropic",
		ProviderMemoryStoreID: "memstore_1",
		Name:                  "memory",
	}
	require.NoError(t, models.CreateAgentMemoryStoreInTransaction(database.Conn(), store))
	require.NoError(t, database.Conn().Unscoped().Delete(&models.Canvas{}, "id = ?", canvas.ID).Error)

	var count int64
	require.NoError(t, database.Conn().Model(&models.AgentMemoryStore{}).Where("canvas_id = ?", canvas.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestAgentMemoryStoreAllowsDifferentCanvasScopes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	firstCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	secondCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	for _, canvasID := range []uuid.UUID{firstCanvas.ID, secondCanvas.ID} {
		require.NoError(t, models.CreateAgentMemoryStoreInTransaction(database.Conn(), &models.AgentMemoryStore{
			OrganizationID:        r.Organization.ID,
			UserID:                r.User,
			CanvasID:              canvasID,
			Provider:              "anthropic",
			ProviderMemoryStoreID: "memstore_" + canvasID.String(),
			Name:                  "memory",
		}))
	}
}
