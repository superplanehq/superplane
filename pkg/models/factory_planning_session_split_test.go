package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactoryPlanningSession_CreateSplitTask(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-split")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	reviewer := createOrgUser(t, org.ID, "split")
	parent, err := factoryModel.CreateWorkOrder(db, "Duplicate a task", "Copy and run with another model.", &userID, []uuid.UUID{userID, reviewer.ID}, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: parent.ID,
	})
	require.NoError(t, err)
	confirmer := createOrgUser(t, org.ID, "split")
	require.NoError(t, session.SendUserMessage(db, "Yes, split it.", confirmer.ID))

	created, err := session.CreateSplitTask(db, factoryModel, PlanningSplitTask{
		Title:       "  Duplicate action on the card  ",
		Description: "Add a duplicate action that opens a new draft.",
	}, uuid.Nil)
	require.NoError(t, err)

	assert.Equal(t, "Duplicate action on the card", created.Title)
	assert.Equal(t, FactoryWorkOrderStateDraft, created.State)
	require.NotNil(t, created.CreatedByID)
	assert.Equal(t, confirmer.ID, *created.CreatedByID, "the person who confirmed the split is the creator")

	assignees, err := created.ListAssignees(db)
	require.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(assignees))
	for _, assignee := range assignees {
		ids = append(ids, assignee.UserID)
	}
	assert.ElementsMatch(t, []uuid.UUID{userID, reviewer.ID, confirmer.ID}, ids, "owners of the parent plus the confirmer")

	split, err := session.SplitTaskOrders(db)
	require.NoError(t, err)
	require.Len(t, split, 1, "the parent draft is linked too, but is not a split task")
	assert.Equal(t, created.ID, split[0].ID)

	require.Len(t, session.Messages, 2)
	message := session.Messages[1]
	assert.Equal(t, PlanningSessionMessageRoleTask, message.Role)
	assert.True(t, message.Delivered)
	var body PlanningTaskMessage
	require.NoError(t, json.Unmarshal([]byte(message.Text), &body))
	assert.Equal(t, created.ID.String(), body.WorkOrderID)
	assert.Equal(t, factoryModel.WorkOrderKey(created.Number), body.Key)
	assert.Equal(t, "Duplicate action on the card", body.Title)
}

func TestFactoryPlanningSession_CreateSplitTaskNeedsAUserReply(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-split-confirm")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	parent, err := factoryModel.CreateWorkOrder(db, "Duplicate a task", "Copy it.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: parent.ID,
	})
	require.NoError(t, err)

	// The first, automatic turn has no user reply, so nothing can confirm a split.
	_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "Part two", Description: "The rest."}, uuid.Nil)
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.ErrorContains(t, err, "user has not replied")
	split, err := session.SplitTaskOrders(db)
	require.NoError(t, err)
	assert.Empty(t, split)

	// A survey answer or chat message from someone without an id still counts as a reply.
	require.NoError(t, session.SendUserMessage(db, "Split it? Yes", uuid.Nil))
	created, err := session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "Part two", Description: "The rest."}, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, created.CreatedByID)
	assert.Equal(t, userID, *created.CreatedByID, "falls back to the parent creator when the reply has no user")
	assert.Equal(t, []uuid.UUID{userID}, created.AssigneeIDs())
}

func TestFactoryPlanningSession_CreateSplitTaskRejectsBadInput(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-split-invalid")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	parent, err := factoryModel.CreateWorkOrder(db, "Duplicate a task", "Copy it.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: parent.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.SendUserMessage(db, "Yes, split it.", uuid.Nil))

	_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: " ", Description: "x"}, uuid.Nil)
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "Part", Description: " "}, uuid.Nil)
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)

	for i := 0; i < maxPlanningSplitTasks; i++ {
		_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "Part", Description: "Some part."}, uuid.Nil)
		require.NoError(t, err)
	}
	_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "One too many", Description: "Nope."}, uuid.Nil)
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)

	require.NoError(t, session.End(db))
	_, err = session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "After end", Description: "Nope."}, uuid.Nil)
	require.ErrorIs(t, err, ErrFactoryPlanningSessionEnded)
}
