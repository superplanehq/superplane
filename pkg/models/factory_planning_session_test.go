package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestTaskCreationSessionRequiresCreator(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, _, factoryModel := setupFactoryWithUser(t, "plan-creator")
	db := database.DB(t.Context())

	err := db.Create(&FactoryPlanningSession{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		FactoryID:      factoryModel.ID,
		Repository:     "acme/payments",
		Kind:           PlanningSessionKindTaskCreation,
		State:          PlanningSessionStateRunning,
		HeartbeatAt:    time.Now(),
	}).Error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory_planning_sessions_task_creation_creator_check")
}

func TestFactorySoftDelete_EndsOpenPlanningSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-delete")
	other := startTestPlanningSession(t, "plan-delete-other")
	db := database.DB(t.Context())
	require.NoError(t, session.BeginWait(db))
	creator := createOrgUser(t, session.OrganizationID, "plan-delete-creator")
	now := time.Now()
	taskSession := &FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  session.OrganizationID,
		FactoryID:       session.FactoryID,
		CreatedByUserID: &creator.ID,
		Repository:      "acme/payments",
		Kind:            PlanningSessionKindTaskCreation,
		State:           PlanningSessionStateRunning,
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, db.Create(taskSession).Error)

	factoryModel, err := FindFactory(db, session.OrganizationID, session.FactoryID)
	require.NoError(t, err)
	require.NoError(t, factoryModel.SoftDelete(db))

	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateEnded, reloaded.State)
	assert.Equal(t, PlanningWaitKindEnded, reloaded.WaitKind)
	require.NotNil(t, reloaded.EndedAt)

	endedTask, err := FindPlanningSession(db, taskSession.OrganizationID, taskSession.FactoryID, taskSession.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateEnded, endedTask.State)

	kept, err := FindPlanningSession(db, other.OrganizationID, other.FactoryID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateRunning, kept.State)
}

func TestFactoryPlanningSession_HeartbeatAndEnd(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-hb")
	db := database.DB(t.Context())

	before := session.HeartbeatAt
	require.NoError(t, session.Heartbeat(db))
	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.HeartbeatAt.After(before))

	require.NoError(t, session.End(db))
	assert.Equal(t, PlanningSessionStateEnded, session.State)
	require.NotNil(t, session.EndedAt)

	err = session.Heartbeat(db)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionEnded)
}

func TestFactoryPlanningSession_EndIfStale(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-stale")
	db := database.DB(t.Context())

	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-6*time.Minute)).Error)
	require.NoError(t, db.First(session, "id = ?", session.ID).Error)

	ended, err := session.EndIfStale(db, time.Now())
	require.NoError(t, err)
	assert.True(t, ended)
	assert.Equal(t, PlanningSessionStateEnded, session.State)
}

func TestFactoryPlanningSession_EndIfStale_KeepsRecentHeartbeat(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-fresh-hb")
	db := database.DB(t.Context())

	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-2*time.Minute)).Error)
	require.NoError(t, db.First(session, "id = ?", session.ID).Error)

	ended, err := session.EndIfStale(db, time.Now())
	require.NoError(t, err)
	assert.False(t, ended)
	assert.Equal(t, PlanningSessionStateRunning, session.State)
}

func TestFactoryPlanningSession_SendMessageResolvesWait(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-wait")
	db := database.DB(t.Context())

	require.NoError(t, session.BeginWait(db))
	assert.Equal(t, PlanningWaitPending, session.WaitState)

	require.NoError(t, session.SendUserMessage(db, "Add refund retries", uuid.Nil))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)
	assert.Equal(t, PlanningWaitKindMessage, session.Wait().Kind)
	assert.Equal(t, "Add refund retries", lastTextMessage(session, PlanningSessionMessageRoleUser))

	result, err := session.ConsumeWait(db)
	require.NoError(t, err)
	assert.Equal(t, PlanningWaitKindMessage, result.Kind)
	assert.Equal(t, "Add refund retries", result.Text)
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.RestoreWait(db, result))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)
	assert.Equal(t, PlanningWaitKindMessage, session.Wait().Kind)
	assert.Equal(t, "Add refund retries", session.Wait().Text)
}

func TestFactoryPlanningSession_SendUserMessageStoresUserID(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-sender")
	db := database.DB(t.Context())
	sender := createOrgUser(t, session.OrganizationID, "plan-sender-user")

	require.NoError(t, session.SendUserMessage(db, "Add refund retries", sender.ID))
	require.Len(t, session.Messages, 1)
	require.NotNil(t, session.Messages[0].UserID)
	assert.Equal(t, sender.ID, *session.Messages[0].UserID)

	require.NoError(t, session.SendUserMessage(db, "Skip the color field", uuid.Nil))
	require.Len(t, session.Messages, 2)
	assert.Nil(t, session.Messages[1].UserID)
}

func TestFactoryPlanningSession_RestoreWaitKeepsQueuedUserMessage(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-restore-queue")
	db := database.DB(t.Context())

	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "Add refund retries", uuid.Nil))
	result, err := session.ConsumeWait(db)
	require.NoError(t, err)
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.SendUserMessage(db, "Add a puppy color field", uuid.Nil))
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.RestoreWait(db, result))
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.BeginWait(db))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)
	assert.Equal(t, PlanningWaitKindMessage, session.Wait().Kind)
	assert.Equal(t, "Add a puppy color field", session.Wait().Text)
}

func TestFactoryPlanningSession_BeginWaitDeliversQueuedUserMessage(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-queue")
	db := database.DB(t.Context())

	require.NoError(t, session.SendUserMessage(db, "Add a puppy color field", uuid.Nil))
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.BeginWait(db))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)

	result, err := session.ConsumeWait(db)
	require.NoError(t, err)
	assert.Equal(t, PlanningWaitKindMessage, result.Kind)
	assert.Equal(t, "Add a puppy color field", result.Text)
	assert.Equal(t, PlanningWaitIdle, session.WaitState)
}

func TestFactoryPlanningSession_ProposeSurveyAndSendClearsIt(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-survey")
	db := database.DB(t.Context())

	require.NoError(t, session.ProposeSurvey(db, PlanningSessionSurvey{
		Questions: []PlanningSessionSurveyQuestion{
			{Prompt: "What is the priority?", Options: []string{"High", "Low"}},
			{Prompt: "What is the scope?", Options: []string{"One file", "The service"}},
		},
	}))
	assert.Equal(t, "What is the priority?", session.CurrentSurvey().Questions[0].Prompt)

	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "Priority: High\nScope: skipped", uuid.Nil))
	assert.Empty(t, session.CurrentSurvey().Questions)
	assert.Equal(t, PlanningWaitKindMessage, session.Wait().Kind)
	assert.Equal(t, "Priority: High\nScope: skipped", session.Wait().Text)
}

func TestFactoryPlanningSession_ProposeSurveyRejectsEmptyQuestions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-survey-empty")
	db := database.DB(t.Context())

	err := session.ProposeSurvey(db, PlanningSessionSurvey{})
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Empty(t, session.CurrentSurvey().Questions)
}

func TestFactoryPlanningSession_ProposeSurveyReplacesPrevious(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-survey-replace")
	db := database.DB(t.Context())

	require.NoError(t, session.ProposeSurvey(db, PlanningSessionSurvey{
		Questions: []PlanningSessionSurveyQuestion{
			{Prompt: "Old?", Options: []string{"A", "B"}},
		},
	}))
	require.NoError(t, session.ProposeSurvey(db, PlanningSessionSurvey{
		Questions: []PlanningSessionSurveyQuestion{
			{Prompt: "New?", Options: []string{"Yes", "No"}},
		},
	}))
	assert.Len(t, session.CurrentSurvey().Questions, 1)
	assert.Equal(t, "New?", session.CurrentSurvey().Questions[0].Prompt)
}

func TestFactoryPlanningSession_RefineNoteLeavesOrdinaryChatAlone(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-refine-chat")
	db := database.DB(t.Context())
	original := session.Draft().WorkOrderID

	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "Refine checkout: please.", uuid.Nil))
	assert.Equal(t, "Refine checkout: please.", lastTextMessage(session, PlanningSessionMessageRoleUser))
	assert.Equal(t, "Refine checkout: please.", session.Wait().Text)
	assert.Equal(t, original, session.Draft().WorkOrderID)
}

func TestFactoryPlanningSession_RefineNoteAttachesExistingBacklogDraft(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-refine-backlog")
	db := database.DB(t.Context())
	factoryModel, err := FindFactory(db, session.OrganizationID, session.FactoryID)
	require.NoError(t, err)
	creator := createOrgUser(t, session.OrganizationID, "plan-refine-backlog-user")

	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &creator.ID, nil, nil)
	require.NoError(t, err)

	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, PlanningRefineNote(factoryModel.WorkOrderKey(order.Number), order.Title), uuid.Nil))
	assert.Equal(t, order.Title, session.Draft().Title)
	assert.Equal(t, order.Description, session.Draft().Description)
	assert.Equal(t, order.ID.String(), session.Draft().WorkOrderID)
}

func TestFactoryPlanningSession_RefineNoteIgnoresOpenWorkOrder(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-refine-open")
	db := database.DB(t.Context())
	factoryModel, err := FindFactory(db, session.OrganizationID, session.FactoryID)
	require.NoError(t, err)
	creator := createOrgUser(t, session.OrganizationID, "plan-refine-open-user")
	original := session.Draft().WorkOrderID

	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &creator.ID, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
		Actor:   &creator.ID,
	})
	require.NoError(t, err)

	note := PlanningRefineNote(factoryModel.WorkOrderKey(order.Number), order.Title)
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, note, uuid.Nil))
	assert.Equal(t, note, session.Wait().Text)
	assert.Equal(t, original, session.Draft().WorkOrderID)
}

func TestEndPlanningSessionForFinishedRun(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-run-pass")
	db := database.DB(t.Context())
	require.NotNil(t, session.CanvasRunID)

	require.NoError(t, EndPlanningSessionForFinishedRun(db, *session.CanvasRunID, CanvasRunResultPassed))
	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateEnded, reloaded.State)
	require.NotNil(t, reloaded.EndedAt)
}

func TestEndPlanningSessionForFinishedRun_Cancelled(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-run-cancel")
	db := database.DB(t.Context())
	require.NotNil(t, session.CanvasRunID)

	require.NoError(t, EndPlanningSessionForFinishedRun(db, *session.CanvasRunID, CanvasRunResultCancelled))
	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateEnded, reloaded.State)
}

func TestEndPlanningSessionForFinishedRun_AlreadyEnded(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-run-ended")
	db := database.DB(t.Context())
	require.NoError(t, session.End(db))
	require.NotNil(t, session.CanvasRunID)

	require.NoError(t, EndPlanningSessionForFinishedRun(db, *session.CanvasRunID, CanvasRunResultFailed))
	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, PlanningSessionStateEnded, reloaded.State)
}

func TestEndPlanningSessionForFinishedRun_NoSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	require.NoError(t, EndPlanningSessionForFinishedRun(database.DB(t.Context()), uuid.New(), CanvasRunResultFailed))
}

func TestFactoryPlanningSession_ListStaleOpenPlanningSessions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := insertTaskCreationSession(t, "plan-list-stale")
	fresh := insertTaskCreationSession(t, "plan-list-fresh")
	db := database.DB(t.Context())

	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-6*time.Minute)).Error)

	stale, err := ListStaleOpenPlanningSessions(db, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, stale, 1)
	assert.Equal(t, session.ID, stale[0].ID)
	assert.NotEqual(t, fresh.ID, stale[0].ID)
}

func TestPlanningSessionMessage_ClearsUserIDOnUserDelete(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-msg-user-delete")
	db := database.DB(t.Context())
	sender := createOrgUser(t, session.OrganizationID, "plan-msg-sender")

	require.NoError(t, session.SendUserMessage(db, "Keep the current retry form.", sender.ID))
	require.Len(t, session.Messages, 1)
	require.NotNil(t, session.Messages[0].UserID)
	assert.Equal(t, sender.ID, *session.Messages[0].UserID)

	require.NoError(t, db.Unscoped().Delete(&User{}, "id = ?", sender.ID).Error)

	messages, err := ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "Keep the current retry form.", messages[0].Text)
	assert.Nil(t, messages[0].UserID)
}

func startTestPlanningSession(t *testing.T, prefix string) *FactoryPlanningSession {
	t.Helper()
	org, userID, factoryModel := setupFactoryWithUser(t, prefix)
	db := database.DB(t.Context())
	canvas, _ := createPlanningCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	return session
}

func insertTaskCreationSession(t *testing.T, prefix string) *FactoryPlanningSession {
	t.Helper()
	org, userID, factoryModel := setupFactoryWithUser(t, prefix)
	db := database.DB(t.Context())
	now := time.Now()
	session := &FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  org.ID,
		FactoryID:       factoryModel.ID,
		CreatedByUserID: &userID,
		Repository:      "acme/payments",
		Kind:            PlanningSessionKindTaskCreation,
		State:           PlanningSessionStateRunning,
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, db.Create(session).Error)
	return session
}

func createPlanningCanvas(t *testing.T, orgID, factoryID, userID uuid.UUID) (*Canvas, string) {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	entrypoint := "start"
	canvas := &Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           "planning-" + uuid.NewString(),
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     entrypoint,
			Name:       "Planning",
			Type:       NodeTypeTrigger,
			State:      CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(NodeRef{
				Trigger: &TriggerRef{Name: "onRun"},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &userID,
			Nodes: datatypes.NewJSONSlice([]Node{{
				ID:   entrypoint,
				Name: "Planning",
				Type: NodeTypeTrigger,
				Ref:  NodeRef{Trigger: &TriggerRef{Name: "onRun"}},
			}}),
			Edges:     datatypes.NewJSONSlice([]Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas, entrypoint
}

func lastTextMessage(session *FactoryPlanningSession, role string) string {
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role == role {
			return session.Messages[i].Text
		}
	}
	return ""
}
