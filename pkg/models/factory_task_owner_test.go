package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestListTaskOwnerCandidates(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	org, _, factoryModel := setupFactoryWithUser(t, "owner-candidates")
	busy := createTaskOwnerMember(t, org.ID, "Busy Person")
	linkTaskOwnerGitHub(t, busy, "old-login")
	linkTaskOwnerGitHub(t, busy, "Busy-Login")
	free := createTaskOwnerMember(t, org.ID, "Free Person")
	linkTaskOwnerGitHub(t, free, "free-login")
	unlinked := createTaskOwnerMember(t, org.ID, "No GitHub")
	removed := createTaskOwnerMember(t, org.ID, "Removed Person")
	linkTaskOwnerGitHub(t, removed, "removed-login")
	require.NoError(t, removed.Delete())

	_, err := factoryModel.CreateWorkOrder(db, "Open draft", "", nil, []uuid.UUID{busy.ID}, nil)
	require.NoError(t, err)
	closed, err := factoryModel.CreateWorkOrder(db, "Closed task", "", nil, []uuid.UUID{busy.ID, free.ID}, nil)
	require.NoError(t, err)
	_, err = closed.UpdateStatus(db, FactoryWorkOrderStatusUpdate{ToState: FactoryWorkOrderStateClosed, Result: FactoryWorkOrderResultRejected})
	require.NoError(t, err)

	candidates, err := ListTaskOwnerCandidates(db, org.ID, factoryModel.ID)
	require.NoError(t, err)

	byName := map[string]TaskOwnerCandidate{}
	for _, candidate := range candidates {
		byName[candidate.Name] = candidate
	}
	require.Len(t, candidates, 4, "the factory user and three members; removed members cannot own tasks")

	assert.Equal(t, 1, byName["Busy Person"].OpenTasks, "closed tasks do not count as load")
	assert.Equal(t, "Busy-Login", byName["Busy Person"].GitHubLogin)
	assert.ElementsMatch(t, []string{"busy-login", "old-login"}, byName["Busy Person"].GitHubLogins)
	assert.Len(t, byName["Busy Person"].GitHubIDs, 2)
	assert.Equal(t, 0, byName["Free Person"].OpenTasks)
	assert.Equal(t, free.ID, byName["Free Person"].UserID)
	assert.Empty(t, byName["No GitHub"].GitHubLogin)
	assert.Empty(t, byName["No GitHub"].GitHubLogins)
	assert.Empty(t, byName["No GitHub"].GitHubIDs)

	found, ok := FindTaskOwnerCandidate(candidates, unlinked.ID.String())
	require.True(t, ok, "a member without GitHub can own a task by user ID")
	assert.Equal(t, unlinked.ID, found.UserID)
	_, ok = FindTaskOwnerCandidate(candidates, "")
	assert.False(t, ok, "an empty login does not match a member without GitHub")

	found, ok = FindTaskOwnerCandidate(candidates, "@OLD-LOGIN")
	require.True(t, ok)
	assert.Equal(t, busy.ID, found.UserID)
	_, ok = FindTaskOwnerCandidate(candidates, "removed-login")
	assert.False(t, ok)
}

func TestFactoryWorkOrder_AssignOwnerIfUnowned(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())

	_, userID, factoryModel := setupFactoryWithUser(t, "assign-owner")

	t.Run("assigns an unowned task and records the reason", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(db, "Intake task", "", nil, nil, nil)
		require.NoError(t, err)

		assigned, err := order.AssignOwnerIfUnowned(db, TaskOwnerAssignment{
			UserID: userID,
			Source: TaskOwnerSourceAgent,
			Reason: "Changed pkg/payments/charge.go 3 times in 30 days.",
		})
		require.NoError(t, err)
		assert.True(t, assigned)

		owners, err := order.ListAssignees(db)
		require.NoError(t, err)
		require.Len(t, owners, 1)
		assert.Equal(t, userID, owners[0].UserID)

		payload := latestAssigneesUpdated(t, order)
		assert.Nil(t, payload.User, "SuperPlane chose the owner, not a person")
		assert.Equal(t, TaskOwnerSourceAgent, payload.Source)
		assert.Equal(t, "Changed pkg/payments/charge.go 3 times in 30 days.", payload.Reason)
		assert.Equal(t, []factory.UserRef{{ID: userID}}, payload.Assigned)
	})

	t.Run("keeps an owner that a person chose", func(t *testing.T) {
		other := createTaskOwnerMember(t, factoryModel.OrganizationID, "Other Person")
		order, err := factoryModel.CreateWorkOrder(db, "Owned task", "", &userID, []uuid.UUID{userID}, nil)
		require.NoError(t, err)

		assigned, err := order.AssignOwnerIfUnowned(db, TaskOwnerAssignment{UserID: other.ID, Source: TaskOwnerSourceAgent})
		require.NoError(t, err)
		assert.False(t, assigned)

		owners, err := order.ListAssignees(db)
		require.NoError(t, err)
		require.Len(t, owners, 1)
		assert.Equal(t, userID, owners[0].UserID)
	})
}

func TestNormalizeTaskOwnerReason(t *testing.T) {
	assert.Equal(t,
		"You changed pkg/auth. Reset your password at now.",
		NormalizeTaskOwnerReason("You changed pkg/auth.\n\nReset your password at https://evil.example/reset now."),
	)
	assert.Equal(t, "Visit today.", NormalizeTaskOwnerReason("Visit www.evil.example today."))
	assert.Empty(t, NormalizeTaskOwnerReason(" \n https://evil.example "))
	assert.Len(t, []rune(NormalizeTaskOwnerReason(strings.Repeat("é", 400))), 280)
}

func createTaskOwnerMember(t *testing.T, orgID uuid.UUID, name string) *User {
	t.Helper()
	email := fmt.Sprintf("owner-%d@example.com", time.Now().UnixNano())
	account, err := CreateAccount(name, email)
	require.NoError(t, err)
	user, err := CreateUser(orgID, account.ID, email, name)
	require.NoError(t, err)
	return user
}

func linkTaskOwnerGitHub(t *testing.T, user *User, login string) {
	t.Helper()
	require.NotNil(t, user.AccountID)
	require.NoError(t, SaveAccountLinkedAccount(
		database.DB(t.Context()),
		NewAccountLinkedAccount(*user.AccountID, ProviderGitHub, uuid.NewString(), login, "", ""),
	))
}

func latestAssigneesUpdated(t *testing.T, order *FactoryWorkOrder) factory.WorkOrderAssigneesUpdated {
	t.Helper()
	events, err := order.ListEvents(database.DB(t.Context()), 20, nil)
	require.NoError(t, err)
	for _, event := range events {
		if event.Type != factory.EventTypeOrderAssigneesUpdated {
			continue
		}
		var payload factory.WorkOrderAssigneesUpdated
		require.NoError(t, json.Unmarshal(event.Data, &payload))
		return payload
	}
	t.Fatal("no assignees updated event")
	return factory.WorkOrderAssigneesUpdated{}
}
