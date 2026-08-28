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

// Regression test for "cannot unassign users": UpdateAssignees must not
// re-insert a stale preloaded `Assignees` association after clearing it.
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

	// Simulate the real call path: a fresh load with `Assignees` preloaded.
	loaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Assignees, 2)

	t.Run("list assignees returns users in assignment order", func(t *testing.T) {
		listed, err := order.ListAssignees(database.Conn())
		require.NoError(t, err)
		require.Len(t, listed, 2)
		assert.Equal(t, userID, listed[0].UserID)
		assert.Equal(t, secondUser.ID, listed[1].UserID)
		require.NotNil(t, listed[0].User)
		require.NotNil(t, listed[1].User)
		assert.Equal(t, secondUser.Name, listed[1].User.Name)
	})

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
