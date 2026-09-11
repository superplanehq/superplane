package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactory_AttachAnalysisSessionReusesEndedSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-reuse")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.SendUserMessage(db, "The retry lives in billing/retry.ts."))
	require.NoError(t, session.End(db))

	nextRun, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	again, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     nextRun.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, session.ID, again.ID)
	assert.Equal(t, PlanningSessionStateRunning, again.State)
	assert.Equal(t, nextRun.ID, *again.CanvasRunID)
	require.Len(t, again.Messages, 1)
	assert.Equal(t, "The retry lives in billing/retry.ts.", again.Messages[0].Text)
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))
	require.NoError(t, session.ProposeConfidence(db, 4, "This issue is a good fit for an agent."))
	require.NoError(t, session.SendUserMessage(db, "Keep the existing retry helper."))

	text, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	assert.Contains(t, text, "Continue this SuperPlane analysis session")
	assert.Contains(t, text, "Stop double charges.")
	assert.Contains(t, text, "4/5")
	assert.Contains(t, text, "This issue is a good fit for an agent.")
	assert.Contains(t, text, "Keep the existing retry helper.")
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, order.ID.String(), session.Draft().WorkOrderID)
	assert.Equal(t, run.ID, *session.CanvasRunID)

	again, err := factoryModel.AttachAnalysisSession(db, AttachAnalysisSessionParams{
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, session.ID, again.ID)
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     otherRun.ID,
		WorkOrderID:     other.ID,
	})
	require.NoError(t, err)
	require.NoError(t, otherSession.ProposeSpec(db, "# Retry invoices\n\n## Executive summary\n\nStop double invoices.\n"))

	otherArtifacts, err := other.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, otherArtifacts, 1)
	assert.Equal(t, planningSpecArtifactKey(other.ID), *otherArtifacts[0].Key)
	assert.Contains(t, string(otherArtifacts[0].Data), "Stop double invoices.")
}

func TestFactoryPlanningSession_ProposeSpecUpdatesTitleOnlyArtifact(t *testing.T) {
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)

	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nFollow-up write.\n"))

	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Nil(t, artifacts[0].Key)
	assert.Contains(t, string(artifacts[0].Data), "Follow-up write.")
}

func TestFactory_MaybeAttachAnalysisSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-analysis-event")
	db := database.DB(t.Context())
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &userID, nil, nil)
	require.NoError(t, err)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)

	event := &CanvasEvent{
		WorkflowID: canvas.ID,
		NodeID:     "start",
		Data: NewJSONValue(map[string]any{
			"type": workOrderCreatedPayloadType,
			"data": map[string]any{
				"workOrder": map[string]any{
					"id":         order.ID.String(),
					"repository": "acme/payments",
				},
			},
		}),
	}
	require.NoError(t, MaybeAttachAnalysisSession(db, canvas, event, run))

	session, err := FindPlanningSessionByRun(db, run.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID.String(), session.Draft().WorkOrderID)

	stale, err := ListStaleOpenPlanningSessions(db, time.Now().Add(10*time.Minute), 10)
	require.NoError(t, err)
	assert.Empty(t, stale)

	open, err := CountOpenPlanningSessions(db, org.ID, factoryModel.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), open)
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
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SendUserMessage(db, "The retry lives in billing/retry.ts."))

	assert.Equal(t, "The retry lives in billing/retry.ts.", session.Wait().Text)
	assert.NotContains(t, session.Wait().Text, "propose_spec")
	assert.NotContains(t, session.Wait().Text, "propose_draft")
}

func createAnalysisCanvas(t *testing.T, orgID, factoryID, userID uuid.UUID) *Canvas {
	t.Helper()
	canvas, _ := createPlanningCanvas(t, orgID, factoryID, userID)
	canvas.Name = "Backlog"
	require.NoError(t, database.DB(t.Context()).Model(canvas).Update("name", canvas.Name).Error)
	return canvas
}
