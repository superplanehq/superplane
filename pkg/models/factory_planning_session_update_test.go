package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactoryPlanningSession_ProposeUpdateWritesScoresSpecAndSurvey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-all")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Clarity: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(3, 2, 3),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
		Survey: &PlanningSessionSurvey{
			Questions: []PlanningSessionSurveyQuestion{{Prompt: "Which service?", Options: []string{"Payments", "Billing"}}},
		},
	})
	require.NoError(t, err)

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	require.Len(t, checks, 3)
	byKey := map[string]FactoryWorkOrderCheck{}
	for _, check := range checks {
		byKey[check.Key] = check
	}
	assert.Equal(t, 3.0, byKey[PlanningClarityCheckKey].Score)
	assert.Equal(t, PlanningClarityCheckName, byKey[PlanningClarityCheckKey].Name)
	assert.Equal(t, 2.0, byKey[PlanningComplexityCheckKey].Score)
	assert.Equal(t, PlanningComplexityCheckName, byKey[PlanningComplexityCheckKey].Name)
	assert.Equal(t, 3.0, byKey[PlanningVerifiabilityCheckKey].Score)
	assert.Equal(t, PlanningVerifiabilityCheckName, byKey[PlanningVerifiabilityCheckKey].Name)
	assert.Equal(t, float64(PlanningReviewScoreMax), byKey[PlanningClarityCheckKey].MaxScore)
	assert.Equal(t, FactoryWorkOrderCheckLevelPositive, byKey[PlanningClarityCheckKey].Level)
	assert.Equal(t, FactoryWorkOrderCheckLevelCaution, byKey[PlanningComplexityCheckKey].Level)

	spec, err := planningSpecBody(db, order)
	require.NoError(t, err)
	assert.Contains(t, spec, "Stop double charges.")

	require.NoError(t, session.reloadMessages(db))
	assert.Equal(t, "Which service?", session.CurrentSurvey().Questions[0].Prompt)
}

func TestFactoryPlanningSession_ProposeUpdateStoresJSONEncodedSpecAsMarkdown(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-encoded-spec")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(3, 3, 3),
		Spec:   `"# Retry refunds\n\nStop double charges.\n"`,
	})
	require.NoError(t, err)

	spec, err := planningSpecBody(db, order)
	require.NoError(t, err)
	assert.Equal(t, "# Retry refunds\n\nStop double charges.", strings.TrimSpace(spec))
	assert.NotContains(t, spec, `\n`)
}

func TestPlanningSpecBodyUnwrapsJSONEncodedArtifact(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-spec-encoded-read")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, _ := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)
	_, err := order.CreateArtifact(db, FactoryWorkOrderArtifactParams{
		Type: FactoryWorkOrderArtifactTypeMarkdown,
		Key:  planningSpecArtifactKey(order.ID),
		Data: map[string]any{
			"name": PlanningSpecArtifactTitle,
			"body": `"# Retry refunds\n\nStop double charges.\n"`,
		},
	})
	require.NoError(t, err)

	spec, err := planningSpecBody(db, order)
	require.NoError(t, err)
	assert.Equal(t, "# Retry refunds\n\nStop double charges.", strings.TrimSpace(spec))
}

func TestFactoryPlanningSession_ProposeUpdateRejectsPartialScores(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-partial")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: &PlanningReviewScores{
			Clarity:    PlanningScoreValue{Score: 3, Summary: "Outcome is clear."},
			Complexity: PlanningScoreValue{Score: 3, Summary: "One run can finish."},
		},
	})
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Contains(t, err.Error(), "verifiability")
}

func TestFactoryPlanningSession_ProposeUpdateRejectsFirstSurveyWithoutScores(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-first-survey")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Survey: &PlanningSessionSurvey{
			Questions: []PlanningSessionSurveyQuestion{{Prompt: "Which service?", Options: []string{"Payments", "Billing"}}},
		},
	})
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Contains(t, err.Error(), "scores")
}

func TestFactoryPlanningSession_ProposeUpdateRejectsNonIntegerScore(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-fraction")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(2.5, 2, 3)})
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Contains(t, err.Error(), "clarity")
}

func TestFactoryPlanningSession_ProposeUpdateAllowsLaterSurveyOnly(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-later-survey")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(1, 2, 3)}))

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Survey: &PlanningSessionSurvey{
			Questions: []PlanningSessionSurveyQuestion{{Prompt: "Which service?", Options: []string{"Payments", "Billing"}}},
		},
	})
	require.NoError(t, err)
	require.NoError(t, session.reloadMessages(db))
	assert.Equal(t, "Which service?", session.CurrentSurvey().Questions[0].Prompt)
}

func TestAnalysisContinuationTextUsesProposeUpdateWhenReviewIsOn(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-continue-review")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(3, 2, 3),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
	}))

	text, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	assert.Contains(t, text, "Call propose_update with scores, spec, and survey in one call")
	assert.Contains(t, text, "When every required score is 3")
	assert.Contains(t, text, "include spec on propose_update")
	assert.Contains(t, text, "Include survey on propose_update")
	assert.Contains(t, text, "Current Clarity: 3/3")
	assert.Contains(t, text, "Current Complexity: 2/3")
	assert.Contains(t, text, "Current Verifiability: 3/3")
	assert.NotContains(t, text, "propose_spec")
	assert.NotContains(t, text, "Call survey only")
}

func TestFactoryPlanningSession_ProposeUpdateIgnoresConfidenceSetting(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-confidence-off")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: false}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(3, 2, 3)}))

	checks, err := order.ListChecks(db)
	require.NoError(t, err)
	assert.Len(t, checks, 3)
}

// Review scoring is not optional, so a stored Confidence=false from the
// legacy settings must not block review auto-start.
func TestWorkOrderReadyForAutoStartReviewRequiresAllThreeThrees(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "auto-start-review")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Clarity: false, Confidence: false}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, &line.ID)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(3, 3, 3),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
	}))

	ready, err := WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.True(t, ready)

	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(3, 3, 2)}))
	ready, err = WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)
}

func TestWorkOrderReadyForAutoStartReviewBlocksLaterMaximumAfterLowFirstScore(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "auto-start-review-low-first")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Clarity: false, Confidence: false}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, &line.ID)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(3, 3, 2),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
	}))

	ready, err := WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)

	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(3, 3, 3)}))
	ready, err = WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)

	require.NoError(t, session.SendUserMessage(db, "The refund path changed.", userID))
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(3, 3, 3)}))
	ready, err = WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)

	nextRun, err := CreateCanvasRunInTransaction(db, canvas.ID, "start", CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, session.AttachAgentRun(db, nextRun.ID, ""))
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(3, 3, 3)}))
	ready, err = WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)
}

func reviewScores(clarity, complexity, verifiability float64) *PlanningReviewScores {
	return &PlanningReviewScores{
		Clarity:       PlanningScoreValue{Score: clarity, Summary: "Outcome, scope, and done are defined."},
		Complexity:    PlanningScoreValue{Score: complexity, Summary: "One agent can finish this in one run."},
		Verifiability: PlanningScoreValue{Score: verifiability, Summary: "Existing tests cover the change."},
	}
}
