package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
)

func TestFactoryPlanningSession_ProposeUpdateWritesScoresSpecAndSurvey(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-update-all")
	db := database.DB(t.Context())
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Clarity: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)

	err := session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(4, 3, 5),
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
	assert.Equal(t, 4.0, byKey[PlanningClarityCheckKey].Score)
	assert.Equal(t, PlanningClarityCheckName, byKey[PlanningClarityCheckKey].Name)
	assert.Equal(t, 3.0, byKey[PlanningComplexityCheckKey].Score)
	assert.Equal(t, PlanningComplexityCheckName, byKey[PlanningComplexityCheckKey].Name)
	assert.Equal(t, 5.0, byKey[PlanningRiskCheckKey].Score)
	assert.Equal(t, FactoryWorkOrderCheckLevelCaution, byKey[PlanningComplexityCheckKey].Level)

	spec, err := planningSpecBody(db, order)
	require.NoError(t, err)
	assert.Contains(t, spec, "Stop double charges.")

	require.NoError(t, session.reloadMessages(db))
	assert.Equal(t, "Which service?", session.CurrentSurvey().Questions[0].Prompt)
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
			Clarity:    PlanningScoreValue{Score: 4, Summary: "Outcome is clear."},
			Complexity: PlanningScoreValue{Score: 4, Summary: "One run can finish."},
		},
	})
	require.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
	assert.Contains(t, err.Error(), "risk")
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

	err := session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(4.5, 3, 5)})
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
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(2, 4, 5)}))

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
	require.NoError(t, EnableExperimentalFeatureInTransaction(db, org.ID, features.FeatureTaskPlanningReview))
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	_, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, nil)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(4, 3, 5),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
	}))

	text, err := AnalysisContinuationText(db, session)
	require.NoError(t, err)
	assert.Contains(t, text, "Call propose_update with scores, spec, and survey in one call")
	assert.Contains(t, text, "include spec on propose_update")
	assert.Contains(t, text, "Include survey on propose_update")
	assert.Contains(t, text, "Current Clarity: 4/5")
	assert.Contains(t, text, "Current Complexity: 3/5")
	assert.Contains(t, text, "Current Risk: 5/5")
	assert.NotContains(t, text, "propose_spec")
	assert.NotContains(t, text, "Call survey only")
}

func TestWorkOrderReadyForAutoStartReviewRequiresAllThreeFives(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "auto-start-review")
	db := database.DB(t.Context())
	require.NoError(t, EnableExperimentalFeatureInTransaction(db, org.ID, features.FeatureTaskPlanningReview))
	require.NoError(t, factoryModel.UpdatePlanning(db, FactoryPlanning{Enabled: true, Clarity: false, Confidence: true}))
	canvas := createAnalysisCanvas(t, org.ID, factoryModel.ID, userID)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	order, session := mustAnalysisOrder(t, db, factoryModel, canvas.ID, userID, &line.ID)
	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{
		Scores: reviewScores(5, 5, 4),
		Spec:   "# Retry refunds\n\nStop double charges.\n",
	}))

	ready, err := WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.False(t, ready)

	require.NoError(t, session.ProposeUpdate(db, PlanningSessionUpdate{Scores: reviewScores(5, 5, 5)}))
	ready, err = WorkOrderReadyForAutoStart(db, factoryModel, order, session)
	require.NoError(t, err)
	assert.True(t, ready)
}

func reviewScores(clarity, complexity, risk float64) *PlanningReviewScores {
	return &PlanningReviewScores{
		Clarity:    PlanningScoreValue{Score: clarity, Summary: "Outcome, scope, and done are defined."},
		Complexity: PlanningScoreValue{Score: complexity, Summary: "One agent can finish this in one run."},
		Risk:       PlanningScoreValue{Score: risk, Summary: "A mistake here is cheap to undo."},
	}
}
