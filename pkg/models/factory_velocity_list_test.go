package models_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListFactoryVelocityPullRequests__LoadsAssigneesInAssignmentOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	first := support.CreateUser(t, r, r.Organization.ID)
	second := support.CreateUser(t, r, r.Organization.ID)

	assigned, err := factory.CreateWorkOrder(db, "Assigned intake", "", nil, nil, nil)
	require.NoError(t, err)
	unassigned, err := factory.CreateWorkOrder(db, "Hand written", "", &r.User, nil, nil)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderAssignee{
		WorkOrderID: assigned.ID,
		UserID:      first.ID,
		CreatedAt:   now.Add(-2 * time.Minute),
	}).Error)
	require.NoError(t, db.Create(&models.FactoryWorkOrderAssignee{
		WorkOrderID: assigned.ID,
		UserID:      second.ID,
		CreatedAt:   now.Add(-time.Minute),
	}).Error)

	mergedAt := now.Add(-time.Hour)
	_, err = assigned.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/11",
		State:    models.FactoryPullRequestStateMerged,
		MergedAt: &mergedAt,
	})
	require.NoError(t, err)
	_, err = unassigned.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/12",
		State:    models.FactoryPullRequestStateMerged,
		MergedAt: &mergedAt,
	})
	require.NoError(t, err)

	rows, err := models.ListFactoryVelocityPullRequests(db, factory.ID, now.Add(-24*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byOrder := map[uuid.UUID]models.FactoryVelocityPullRequest{}
	for _, row := range rows {
		byOrder[row.WorkOrderID] = row
	}

	assignedRow := byOrder[assigned.ID]
	assert.Nil(t, assignedRow.CreatedByID)
	assert.Equal(t, []uuid.UUID{first.ID, second.ID}, assignedRow.AssigneeIDs)

	unassignedRow := byOrder[unassigned.ID]
	require.NotNil(t, unassignedRow.CreatedByID)
	assert.Equal(t, r.User, *unassignedRow.CreatedByID)
	assert.Empty(t, unassignedRow.AssigneeIDs)
}

func Test__ListFactoryVelocityPullRequests__OrdersTiedAssigneesByUserID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	first := support.CreateUser(t, r, r.Organization.ID)
	second := support.CreateUser(t, r, r.Organization.ID)

	order, err := factory.CreateWorkOrder(db, "Tied assignees", "", nil, nil, nil)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderAssignee{
		WorkOrderID: order.ID,
		UserID:      first.ID,
		CreatedAt:   now,
	}).Error)
	require.NoError(t, db.Create(&models.FactoryWorkOrderAssignee{
		WorkOrderID: order.ID,
		UserID:      second.ID,
		CreatedAt:   now,
	}).Error)

	mergedAt := now.Add(-time.Hour)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:      "https://github.com/example/repo/pull/13",
		State:    models.FactoryPullRequestStateMerged,
		MergedAt: &mergedAt,
	})
	require.NoError(t, err)

	rows, err := models.ListFactoryVelocityPullRequests(db, factory.ID, now.Add(-24*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	want := []uuid.UUID{first.ID, second.ID}
	if bytes.Compare(first.ID[:], second.ID[:]) > 0 {
		want = []uuid.UUID{second.ID, first.ID}
	}
	assert.Equal(t, want, rows[0].AssigneeIDs)
}
