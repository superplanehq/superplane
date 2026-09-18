package models

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactory_AttachAnalysisSessionReusesEndedSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-reuse")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.SendUserMessage(db, "The retry lives in billing/retry.ts.", uuid.Nil))
	require.NoError(t, session.End(db))

	nextRun, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	again, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: nextRun.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, session.ID, again.ID)
	assert.Equal(t, PlanningSessionStateRunning, again.State)
	assert.Equal(t, nextRun.ID, *again.CanvasRunID)
	require.Len(t, again.Messages, 1)
	assert.Equal(t, "The retry lives in billing/retry.ts.", again.Messages[0].Text)
}

func TestFactory_AttachAnalysisSessionReconnectsRunningSessionWithoutRun(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-reconnect")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	initialRun, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: initialRun.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.SendUserMessage(db, "Use the existing retry helper.", uuid.Nil))
	require.NoError(t, session.DetachAgentRun(db))

	nextRun, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	reconnected, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: nextRun.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, session.ID, reconnected.ID)
	assert.Equal(t, PlanningSessionStateRunning, reconnected.State)
	require.NotNil(t, reconnected.CanvasRunID)
	assert.Equal(t, nextRun.ID, *reconnected.CanvasRunID)
	require.Len(t, reconnected.Messages, 1)
	assert.Equal(t, "Use the existing retry helper.", reconnected.Messages[0].Text)
}

func TestFactoryPlanningSession_NeedsAnalysisRestart(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-restart")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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
	assert.False(t, session.NeedsAnalysisRestart(db))

	now := time.Now()
	require.NoError(t, db.Model(run).Updates(map[string]any{
		"state":       CanvasRunStateFinished,
		"result":      CanvasRunResultCancelled,
		"finished_at": &now,
		"updated_at":  &now,
	}).Error)
	assert.True(t, session.NeedsAnalysisRestart(db))

	require.NoError(t, session.End(db))
	assert.True(t, session.NeedsAnalysisRestart(db))
}

func TestFactoryWorkOrder_StatusTransitionEndsAnalysisSession(t *testing.T) {
	for _, test := range []struct {
		name              string
		state             string
		result            string
		expectedRunResult string
	}{
		{name: "start", state: FactoryWorkOrderStateOpen, expectedRunResult: CanvasRunResultPassed},
		{name: "archive", state: FactoryWorkOrderStateClosed, result: FactoryWorkOrderResultRejected},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, database.TruncateTables())
			org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-transition-"+test.name)
			db := database.DB(t.Context())
			canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
			order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
			require.NoError(t, err)
			run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
			require.NoError(t, err)
			session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
				Repository:  "acme/payments",
				CanvasID:    canvas.ID,
				CanvasRunID: run.ID,
				WorkOrderID: order.ID,
			})
			require.NoError(t, err)

			changed, err := order.UpdateStatus(db, FactoryWorkOrderStatusUpdate{
				ToState: test.state,
				Result:  test.result,
				Actor:   &userID,
			})
			require.NoError(t, err)
			assert.True(t, changed)

			session, err = FindPlanningSession(db, org.ID, factoryModel.ID, session.ID)
			require.NoError(t, err)
			assert.Equal(t, PlanningSessionStateEnded, session.State)
			run, err = FindUnscopedCanvasRun(db, run.ID)
			require.NoError(t, err)
			assert.Equal(t, CanvasRunStateCancelling, run.State)
			assert.Equal(t, test.expectedRunResult, run.Result)
		})
	}
}

func TestAnalysisContinuationTextIncludesSpecScoreAndChat(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-continue")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))
	require.NoError(t, session.ProposeClarity(db, 4, "One decision is still open."))
	require.NoError(t, session.ProposeConfidence(db, 3, "This issue is a mixed fit for an agent."))
	require.NoError(t, session.RecordAgentMessage(db, "I found the retry seam in billing/retry.go."))
	require.NoError(t, session.SendUserMessage(db, "Keep the existing retry helper.", uuid.Nil))

	text, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	assert.Contains(t, text, "Continue this SuperPlane analysis session")
	assert.Contains(t, text, "Follow the task prompt")
	assert.Contains(t, text, "Call propose_clarity and propose_confidence every turn")
	assert.Contains(t, text, "Do not leave a written plan unpublished")
	assert.NotContains(t, text, "If Clarity is still 1 or 2")
	assert.Contains(t, text, "Stop double charges.")
	assert.Contains(t, text, "Current Clarity: 4/5\nOne decision is still open.")
	assert.Contains(t, text, "Current Confidence: 3/5\nThis issue is a mixed fit for an agent.")
	assert.Contains(t, text, "I found the retry seam in billing/retry.go.")
	assert.Contains(t, text, "Keep the existing retry helper.")
	assert.Equal(t, 1, strings.Count(text, "Keep the existing retry helper."))
}

func TestAnalysisContinuationTextListsSplitTasks(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-split")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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
	require.NoError(t, session.SendUserMessage(db, "Yes, split it.", uuid.Nil))
	created, err := session.CreateSplitTask(db, factoryModel, PlanningSplitTask{Title: "Add the retry table", Description: "Schema only."}, uuid.Nil)
	require.NoError(t, err)

	text, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	key := factoryModel.WorkOrderKey(created.Number)
	assert.Contains(t, text, "SuperPlane: Created task "+key+": Add the retry table")
	assert.Contains(t, text, "Do not create a task that this session already created")
	assert.NotContains(t, text, created.ID.String(), "the raw JSON body stays out of the prompt")
}

func TestAnalysisConversationWindowKeepsRecentMessagesWithHeadroom(t *testing.T) {
	messages := []PlanningSessionMessage{
		{Role: PlanningSessionMessageRoleUser, Text: "oldest context"},
		{Role: PlanningSessionMessageRoleAgent, Text: "older answer"},
		{Role: PlanningSessionMessageRoleUser, Text: "recent context"},
		{Role: PlanningSessionMessageRoleAgent, Text: "latest answer"},
	}
	hardLimit := analysisMessageContextCharacters(messages[1]) +
		analysisMessageContextCharacters(messages[2]) +
		analysisMessageContextCharacters(messages[3]) + 1

	window := analysisConversationWindow(messages, hardLimit)

	require.NotEmpty(t, window.Messages)
	assert.Equal(t, messages[len(messages)-len(window.Messages):], window.Messages)
	assert.Equal(t, len(messages)-len(window.Messages), window.Omitted)
	assert.LessOrEqual(t, analysisMessagesContextCharacters(window.Messages), hardLimit*analysisRewindRetentionPercent/100)
}

func TestAnalysisConversationWindowKeepsAllStoredMessagesWhenTheyFit(t *testing.T) {
	messages := []PlanningSessionMessage{
		{Role: PlanningSessionMessageRoleUser, Text: "first"},
		{Role: PlanningSessionMessageRoleAgent, Text: "second"},
	}

	window := analysisConversationWindow(messages, analysisMessagesContextCharacters(messages))

	assert.Equal(t, messages, window.Messages)
	assert.Zero(t, window.Omitted)
}

func TestAnalysisConversationWindowTruncatesOnlyTheRewindCopyOfAnOversizedLatestMessage(t *testing.T) {
	message := PlanningSessionMessage{Role: PlanningSessionMessageRoleUser, Text: strings.Repeat("context ", 100)}
	messages := []PlanningSessionMessage{message}

	window := analysisConversationWindow(messages, 120)

	require.Len(t, window.Messages, 1)
	assert.Contains(t, window.Messages[0].Text, analysisRewindTruncationMarker)
	assert.Equal(t, message.Text, messages[0].Text)
	assert.LessOrEqual(t, analysisMessagesContextCharacters(window.Messages), 120)
}

func TestAnalysisContinuationTextBoundsRewindWithoutDeletingHistory(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-history")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	for index := range 30 {
		message := fmt.Sprintf("message-%02d %s", index, strings.Repeat("context ", 300))
		if index%2 == 0 {
			require.NoError(t, session.SendUserMessage(db, message, uuid.Nil))
			continue
		}
		require.NoError(t, session.RecordAgentMessage(db, message))
	}

	history, err := ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	require.Len(t, history, 30)
	rewind, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	assert.NotContains(t, rewind, "message-00")
	assert.Contains(t, rewind, "message-29")
	assert.Contains(t, rewind, "older messages are not in this rewind")

	historyAfterRewind, err := ListPlanningSessionMessages(db, session.ID)
	require.NoError(t, err)
	assert.Equal(t, history, historyAfterRewind)
}

func TestFactory_AttachAnalysisSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-attach")
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
	assert.Equal(t, order.ID.String(), session.Draft().WorkOrderID)
	assert.Equal(t, run.ID, *session.CanvasRunID)

	again, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, session.ID, again.ID)
}

func TestFactoryPlanningSession_ProposeConfidenceWithoutSpec(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-score-only")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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

	require.NoError(t, session.ProposeClarity(db, 2, "The request is still missing the failing path."))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	assert.Empty(t, artifacts)

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, PlanningClarityCheckKey, checks[0].Key)
	assert.Equal(t, PlanningClarityCheckName, checks[0].Name)
	assert.Equal(t, 2.0, checks[0].Score)
	assert.Equal(t, FactoryWorkOrderCheckLevelCritical, checks[0].Level)
	assert.Equal(t, "The request is still missing the failing path.", checks[0].Summary)
}

func TestFactoryPlanningSession_ProposeClarityAndConfidenceAreSeparateChecks(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-two-scores")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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

	require.NoError(t, session.ProposeClarity(db, 5, "The plan is ready."))
	require.NoError(t, session.ProposeConfidence(db, 2, "The change touches billing and has no test to prove it."))
	require.NoError(t, session.ProposeConfidence(db, 3, "A split into two tasks would help."))

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 2)
	byKey := map[string]FactoryWorkOrderCheck{}
	for _, check := range checks {
		byKey[check.Key] = check
	}
	assert.Equal(t, 5.0, byKey[PlanningClarityCheckKey].Score)
	assert.Equal(t, "The plan is ready.", byKey[PlanningClarityCheckKey].Summary)
	assert.Equal(t, 3.0, byKey[PlanningConfidenceCheckKey].Score)
	assert.Equal(t, PlanningConfidenceCheckName, byKey[PlanningConfidenceCheckKey].Name)
	assert.Equal(t, FactoryWorkOrderCheckLevelCaution, byKey[PlanningConfidenceCheckKey].Level)
}

func TestValidatePlanningScoreRejectsZero(t *testing.T) {
	err := validatePlanningScore(planningConfidenceScore, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Contains(t, err.Error(), "confidence score must be 1 through 5")
}

func TestValidatePlanningScoreAcceptsOneAndFive(t *testing.T) {
	require.NoError(t, validatePlanningScore(planningClarityScore, 1))
	require.NoError(t, validatePlanningScore(planningClarityScore, 5))
}

func TestAnalysisConversationWindowSkipsPlanBanners(t *testing.T) {
	messages := []PlanningSessionMessage{
		{Role: PlanningSessionMessageRoleUser, Text: "Add refund retries."},
		{Role: PlanningSessionMessageRolePlan, Text: `{"score":4,"summary":"Clear"}`},
		{Role: PlanningSessionMessageRoleAgent, Text: "I updated the plan."},
	}

	window := analysisConversationWindow(messages, analysisMessagesContextCharacters(messages))

	assert.Equal(t, []PlanningSessionMessage{messages[0], messages[2]}, window.Messages)
	assert.Zero(t, window.Omitted)
	assert.NotContains(t, window.Messages, messages[1])
}

func TestFactoryPlanningSession_ProposeSpecAndConfidence(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-propose")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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

	body := "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"
	require.NoError(t, session.ProposeSpec(db, body))
	require.NoError(t, session.ProposeConfidence(db, 4, "This issue is a good fit for an agent."))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Equal(t, planningSpecArtifactKey(order.ID), *artifacts[0].Key)
	assert.Contains(t, string(artifacts[0].Data), "Stop double charges.")

	require.NoError(t, session.ProposeSpec(db, body+"\n## Problem\n\nThe retry is missing.\n"))
	artifacts, err = order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Contains(t, string(artifacts[0].Data), "The retry is missing.")

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, PlanningConfidenceCheckKey, checks[0].Key)
	assert.Equal(t, 4.0, checks[0].Score)
	assert.Equal(t, FactoryWorkOrderCheckLevelPositive, checks[0].Level)
	assert.Equal(t, "This issue is a good fit for an agent.", checks[0].Summary)

	found, err := FindPlanningSessionByDraftWorkOrder(db, org.ID, factoryModel.ID, order.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)

	other, err := factoryModel.CreateWorkOrder(db, "Retry invoices", "Stop double invoices.", &userID, nil, nil)
	require.NoError(t, err)
	otherRun, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	otherSession, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: otherRun.ID,
		WorkOrderID: other.ID,
	})
	require.NoError(t, err)
	require.NoError(t, otherSession.ProposeSpec(db, "# Retry invoices\n\n## Executive summary\n\nStop double invoices.\n"))

	otherArtifacts, err := other.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, otherArtifacts, 1)
	assert.Equal(t, planningSpecArtifactKey(other.ID), *otherArtifacts[0].Key)
	assert.Contains(t, string(otherArtifacts[0].Data), "Stop double invoices.")
}

func TestFactoryPlanningSession_ProposeSpecDoesNotOverwriteTitleOnlyArtifact(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-title")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	_, err = order.CreateArtifact(db, FactoryWorkOrderArtifactParams{
		Type: FactoryWorkOrderArtifactTypeMarkdown,
		Data: map[string]any{
			"name":  PlanningSpecArtifactTitle,
			"title": PlanningSpecArtifactTitle,
			"body":  "# Retry refunds\n\n## Executive summary\n\nFirst write.\n",
		},
	})
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

	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nFollow-up write.\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)
	for _, artifact := range artifacts {
		if artifact.Key == nil {
			assert.Contains(t, string(artifact.Data), "First write.")
			continue
		}
		assert.Equal(t, planningSpecArtifactKey(order.ID), *artifact.Key)
		assert.Contains(t, string(artifact.Data), "Follow-up write.")
	}
}

func TestFactoryPlanningSession_ProposeSpecRestoresSignedFileURLs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-spec-restore")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	file := mustReadyTaskFile(t, db, org.ID, factoryModel.ID, order.ID, userID, "shot.png", "image/png")
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	gcsURL := "https://storage.googleapis.com/superplane-prod-global/881b70a0-5c9e-47da-a4ca-395f402f3aea/orgs/" +
		org.ID.String() + "/workspaces/" + factoryModel.ID.String() + "/tasks/" + order.ID.String() + "/" + file.ID.String() +
		"?X-Goog-Algorithm=GOOG4-RSA-SHA256&sp_file=1"
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nSee ![shot.png]("+gcsURL+")\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	stored := string(artifacts[0].Data)
	assert.Contains(t, stored, "![shot.png]("+blob.FileRef(file.ID)+")")
	assert.NotContains(t, stored, "sp_file=1")

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, "Stop double charges.", reloaded.Description)
}

func TestFactoryPlanningSession_ProposeSpecAppendsMissingDescriptionFileRefs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-spec-append")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	file := mustReadyTaskFile(t, db, org.ID, factoryModel.ID, order.ID, userID, "shot.png", "image/png")
	description := "See ![shot.png](" + blob.FileRef(file.ID) + ")"
	require.NoError(t, order.UpdateContent(db, nil, &description))
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	stored := string(artifacts[0].Data)
	assert.Contains(t, stored, "![shot.png]("+blob.FileRef(file.ID)+")")

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, description, reloaded.Description)
}

func TestFactoryPlanningSession_ProposeSpecEscapesMarkdownFilenames(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-spec-escape")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	file := mustReadyTaskFile(t, db, org.ID, factoryModel.ID, order.ID, userID, "shot](evil.com", "image/png")
	description := "See ![orig](" + blob.FileRef(file.ID) + ")"
	require.NoError(t, order.UpdateContent(db, nil, &description))
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	stored := string(artifacts[0].Data)
	assert.Contains(t, stored, "![shot__evil.com]("+blob.FileRef(file.ID)+")")
	assert.NotContains(t, stored, "shot](evil.com")
}

func TestFactoryPlanningSession_ProposeSpecDropsForeignSignedURLs(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-spec-drop")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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

	foreign := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	signed := "https://app.example/api/v1/public/files/" + foreign.String() + "?expires=1&sig=abc&sp_file=1"
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nSee ![x]("+signed+")\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	stored := string(artifacts[0].Data)
	assert.NotContains(t, stored, "sp_file=1")
	assert.NotContains(t, stored, blob.FileRef(foreign))
}

func TestFactory_MaybeAttachAnalysisSessionForSystemCreatedWorkOrder(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-event")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	canvas.Name = "Triage incoming tasks"
	require.NoError(t, db.Model(canvas).Update("name", canvas.Name).Error)
	require.NoError(t, EnableExperimentalFeature(org.ID, features.FeatureFactoryCreateWithAgent))
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", nil, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)

	event := &CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     FactoryAppBacklogTriggerID,
		RunID:      run.ID,
		Data: NewJSONValue(map[string]any{
			"type": workOrderCreatedPayloadType,
			"data": map[string]any{
				WorkOrderCreatedRefinementEnabledDataKey: true,
				"workOrder": map[string]any{
					"id":         order.ID.String(),
					"repository": "acme/payments",
				},
			},
		}),
	}
	require.NoError(t, DisableExperimentalFeature(org.ID, features.FeatureFactoryCreateWithAgent))
	require.NoError(t, MaybeAttachAnalysisSession(db, canvas, event, run))
	require.NoError(t, MaybeAttachAnalysisSession(db, canvas, event, run))

	session, err := FindPlanningSessionByRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID.String(), session.Draft().WorkOrderID)
	assert.Equal(t, PlanningSessionKindWorkOrderAnalysis, session.Kind)
	assert.Nil(t, session.CreatedByUserID)
	var sessionCount int64
	require.NoError(t, db.Model(&FactoryPlanningSession{}).
		Where("draft_work_order_id = ? AND kind = ?", order.ID, PlanningSessionKindWorkOrderAnalysis).
		Count(&sessionCount).Error)
	assert.Equal(t, int64(1), sessionCount)

	stale, err := ListStaleOpenPlanningSessions(db, time.Now().Add(10*time.Minute), 10)
	require.NoError(t, err)
	assert.Empty(t, stale)

	open, err := CountOpenPlanningSessions(db, org.ID, factoryModel.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), open)
}

func TestFactory_MaybeAttachAnalysisSessionRequiresFeatureAtCreation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-flag-off")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	event := analysisWorkOrderCreatedEvent(canvas.ID, run.ID, order.ID, false)

	require.NoError(t, MaybeAttachAnalysisSession(db, canvas, event, run))
	_, err = FindPlanningSessionByRun(db, run.ID)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionNotFound)
}

func TestFactory_MaybeAttachAnalysisSessionRejectsCustomWorkOrderCanvas(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-custom")
	db := database.DB(t.Context())
	canvas, _ := createPlanningCanvas(t, org.ID, factoryModel.ID, userID)
	version, err := FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := version.Nodes
	nodes[0].ID = FactoryAppBacklogTriggerID
	nodes[0].Ref = NodeRef{Trigger: &TriggerRef{Name: "onWorkOrder"}}
	require.NoError(t, db.Model(version).Update("nodes", nodes).Error)
	createBacklogTriggerCanvasNode(t, db, canvas.ID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, FactoryAppBacklogTriggerID, CanvasRunStateStarted, "")
	require.NoError(t, err)
	event := analysisWorkOrderCreatedEvent(canvas.ID, run.ID, order.ID, true)

	require.NoError(t, MaybeAttachAnalysisSession(db, canvas, event, run))
	_, err = FindPlanningSessionByRun(db, run.ID)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionNotFound)
}

func TestFactoryPlanningSession_AnalysisFollowUpKeepsTheRequest(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-follow")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
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
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "The retry lives in billing/retry.ts.", uuid.Nil))

	assert.Equal(t, "The retry lives in billing/retry.ts.", session.Wait().Text)
	assert.NotContains(t, session.Wait().Text, "propose_spec")
	assert.NotContains(t, session.Wait().Text, "propose_draft")
}

func mustReadyTaskFile(
	t *testing.T,
	db *gorm.DB,
	organizationID, factoryID, workOrderID, userID uuid.UUID,
	filename, contentType string,
) *File {
	t.Helper()
	file, err := CreatePendingFile(db, CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		WorkOrderID:    workOrderID,
		Filename:       filename,
		ContentType:    contentType,
		CreatedByID:    userID,
	})
	require.NoError(t, err)
	require.NoError(t, file.MarkReady(db, 12, "abc"))
	return file
}

func createAnalysisCanvas(t *testing.T, orgID, factoryID, userID uuid.UUID) *Canvas {
	t.Helper()
	canvas, _ := createPlanningCanvas(t, orgID, factoryID, userID)
	canvas.Name = "Backlog"
	db := database.DB(t.Context())
	require.NoError(t, db.Model(canvas).Update("name", canvas.Name).Error)
	version, err := FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := version.Nodes
	nodes[0].ID = FactoryAppBacklogTriggerID
	nodes[0].Ref = NodeRef{Trigger: &TriggerRef{Name: "onWorkOrder"}}
	nodes[0].Metadata = FactoryAppTemplateMetadata(FactoryAppTemplateBacklogID, 1)
	require.NoError(t, db.Model(version).Update("nodes", nodes).Error)
	createBacklogTriggerCanvasNode(t, db, canvas.ID)
	return canvas
}

func createBacklogTriggerCanvasNode(t *testing.T, db *gorm.DB, canvasID uuid.UUID) {
	t.Helper()
	var node CanvasNode
	require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvasID, "start").First(&node).Error)
	node.NodeID = FactoryAppBacklogTriggerID
	node.Name = "On Task"
	node.Ref = datatypes.NewJSONType(NodeRef{Trigger: &TriggerRef{Name: "onWorkOrder"}})
	require.NoError(t, db.Create(&node).Error)
}

func analysisWorkOrderCreatedEvent(canvasID, runID, workOrderID uuid.UUID, refinementEnabled bool) *CanvasEvent {
	return &CanvasEvent{
		WorkflowID: canvasID,
		NodeID:     FactoryAppBacklogTriggerID,
		RunID:      runID,
		Data: NewJSONValue(map[string]any{
			"type": workOrderCreatedPayloadType,
			"data": map[string]any{
				WorkOrderCreatedRefinementEnabledDataKey: refinementEnabled,
				"workOrder": map[string]any{
					"id":         workOrderID.String(),
					"repository": "acme/payments",
				},
			},
		}),
	}
}
