package models

import (
	"encoding/json"
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

func TestFactoryWorkOrder_UpdateAssigneesUnassignsAll(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, userID, factoryModel := setupFactoryWithUser(t, "unassign")

	secondAccount, err := CreateAccount("Unassign Second User", "unassign-second@example.com")
	require.NoError(t, err)
	secondUser, err := CreateUser(organization.ID, secondAccount.ID, secondAccount.Email, secondAccount.Name)
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(
		database.Conn(),
		"Unassign me",
		"",
		&userID,
		[]uuid.UUID{userID, secondUser.ID},
		nil,
	)
	require.NoError(t, err)

	reloaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	require.Len(t, reloaded.Assignees, 2)

	require.NoError(t, reloaded.UpdateAssignees(database.Conn(), []uuid.UUID{}, userID))

	reloaded, err = factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	assert.Empty(t, reloaded.Assignees)

	events, err := reloaded.ListEvents(database.Conn(), 10, nil)
	require.NoError(t, err)
	assigneesEvent := findEventOfType(t, events, factory.EventTypeOrderAssigneesUpdated)
	var payload factory.WorkOrderAssigneesUpdated
	require.NoError(t, json.Unmarshal(assigneesEvent.Data, &payload))
	assert.Empty(t, payload.Assigned)
	assert.ElementsMatch(t, []factory.UserRef{{ID: userID}, {ID: secondUser.ID}}, payload.Unassigned)
}
