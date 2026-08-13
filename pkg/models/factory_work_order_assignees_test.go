package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestAssigneeDiff(t *testing.T) {
	kept := uuid.New()
	added := uuid.New()
	removed := uuid.New()

	assigned, unassigned := assigneeDiff(
		[]uuid.UUID{kept, removed},
		[]uuid.UUID{kept, added},
	)

	requireLen := func(t *testing.T, refs []factory.UserRef, n int) {
		t.Helper()
		assert.Len(t, refs, n)
	}

	requireLen(t, assigned, 1)
	assert.Equal(t, added, assigned[0].ID)
	requireLen(t, unassigned, 1)
	assert.Equal(t, removed, unassigned[0].ID)
}

func TestAssigneeDiffNoChanges(t *testing.T) {
	assignee := uuid.New()

	assigned, unassigned := assigneeDiff([]uuid.UUID{assignee}, []uuid.UUID{assignee})

	assert.Empty(t, assigned)
	assert.Empty(t, unassigned)
}

// Regression test for "cannot unassign users": UpdateAssignees is normally
// called on a *FactoryWorkOrder that still has its old `Assignees`
// association preloaded from before the update (e.g. loaded via
// Factory.FindWorkOrder). GORM's Update callbacks save has-many associations
// present on the model by default, so a naive `tx.Model(o).Update(...)` for
// an unrelated column (updated_at) would silently re-insert that stale,
// preloaded assignee list right after ReplaceAssignees deleted it — making
// it look like unassigning (or partially reassigning) never took effect.
func TestFactoryWorkOrder_UpdateAssignees(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	_, userID, factoryModel := setupFactoryWithUser(t, "update-assignees")

	account, err := CreateAccount("Second User", "update-assignees-second-user@example.com")
	require.NoError(t, err)
	secondUser, err := CreateUser(factoryModel.OrganizationID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Ship it", "", &userID, nil, nil)
	require.NoError(t, err)

	require.NoError(t, order.UpdateAssignees(database.Conn(), []uuid.UUID{userID, secondUser.ID}, userID))

	// Simulate the real call path: a fresh load with `Assignees` preloaded,
	// then a call to UpdateAssignees on that (now slightly stale-by-the-time-
	// it-writes) model.
	loaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Assignees, 2)

	t.Run("unassigning everyone actually clears all assignees", func(t *testing.T) {
		require.NoError(t, loaded.UpdateAssignees(database.Conn(), nil, userID))

		refreshed, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
		require.NoError(t, err)
		assert.Empty(t, refreshed.Assignees)
		assert.Empty(t, loaded.Assignees, "in-memory association should also be cleared")
	})

	t.Run("partial removal only keeps the requested assignee", func(t *testing.T) {
		require.NoError(t, loaded.UpdateAssignees(database.Conn(), []uuid.UUID{userID, secondUser.ID}, userID))

		reloaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
		require.NoError(t, err)
		require.Len(t, reloaded.Assignees, 2)

		require.NoError(t, reloaded.UpdateAssignees(database.Conn(), []uuid.UUID{userID}, userID))

		refreshed, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
		require.NoError(t, err)
		require.Len(t, refreshed.Assignees, 1)
		assert.Equal(t, userID, refreshed.Assignees[0].UserID)
	})
}
