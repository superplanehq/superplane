package humanreview

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
)

// This boundary owns approval policy. A later comment must not silently clear
// a change request, and an approval on an older commit must never pass the gate.
func TestEvaluateGate(t *testing.T) {
	now := time.Now()
	required := true
	input := GateInput{
		Enabled: true, AssessmentValid: true, DecisionValid: true, HeadSHA: "current",
		Decision: Decision{AssessedCommit: "current", ReviewRequired: &required, Rationale: "Review the database change.", SelectedReviewers: []Reviewer{{ID: 1, Login: "reviewer", Reason: "Owns the database."}}},
		Eligible: map[int64]bool{1: true, 2: true},
	}
	for _, test := range []struct {
		name            string
		reviews         []Review
		assessmentValid bool
		conclusion      string
	}{
		{name: "missing approval", assessmentValid: true, conclusion: "failure"},
		{name: "current selected approval", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now}}, conclusion: "success"},
		{name: "old commit approval", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "APPROVED", CommitID: "old", SubmittedAt: now}}, conclusion: "failure"},
		{name: "dismissed approval", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "DISMISSED", CommitID: "current", SubmittedAt: now}}, conclusion: "failure"},
		{name: "unselected change request", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now}, {ID: 11, UserID: 2, State: "CHANGES_REQUESTED", CommitID: "old", SubmittedAt: now}}, conclusion: "failure"},
		{name: "comment leaves change request", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "CHANGES_REQUESTED", CommitID: "current", SubmittedAt: now}, {ID: 11, UserID: 1, State: "COMMENTED", CommitID: "current", SubmittedAt: now.Add(time.Minute)}}, conclusion: "failure"},
		{name: "later approval clears changes", assessmentValid: true, reviews: []Review{{ID: 10, UserID: 1, State: "CHANGES_REQUESTED", CommitID: "current", SubmittedAt: now}, {ID: 11, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now.Add(time.Minute)}}, conclusion: "success"},
		{name: "out of order reviews retain latest disposition", assessmentValid: true, reviews: []Review{{ID: 11, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now.Add(time.Minute)}, {ID: 10, UserID: 1, State: "CHANGES_REQUESTED", CommitID: "current", SubmittedAt: now}}, conclusion: "success"},
		{name: "approval cannot repair assessment", reviews: []Review{{ID: 10, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now}}, conclusion: "failure"},
	} {
		t.Run(test.name, func(t *testing.T) {
			current := input
			current.AssessmentValid = test.assessmentValid
			current.Reviews = test.reviews
			require.Equal(t, test.conclusion, EvaluateGate(current).Conclusion)
		})
	}
	input.Eligible[1] = false
	input.Reviews = []Review{{ID: 10, UserID: 1, State: "APPROVED", CommitID: "current", SubmittedAt: now}}
	require.Equal(t, "failure", EvaluateGate(input).Conclusion, "lost write access invalidates approval")
	input.Enabled = false
	require.Equal(t, "success", EvaluateGate(input).Conclusion)
}

func TestNoReviewDecisionStillHonorsChangesRequested(t *testing.T) {
	required := false
	input := GateInput{Enabled: true, AssessmentValid: true, DecisionValid: true, HeadSHA: "head", Decision: Decision{AssessedCommit: "head", ReviewRequired: &required, Rationale: "Every check is favorable."}, Eligible: map[int64]bool{2: true}}
	require.Equal(t, "success", EvaluateGate(input).Conclusion)
	input.Reviews = []Review{{ID: 1, UserID: 2, State: "CHANGES_REQUESTED", SubmittedAt: time.Now()}}
	require.Equal(t, "failure", EvaluateGate(input).Conclusion)
}

func TestValidateDecisionRejectsIncompleteOrStaleOutput(t *testing.T) {
	required := false
	valid := Decision{AssessedCommit: "head", ReviewRequired: &required, Rationale: "All findings are favorable."}
	require.NoError(t, ValidateDecision(valid, "head"))
	require.Error(t, ValidateDecision(valid, "new-head"))
	valid.ReviewRequired = nil
	require.Error(t, ValidateDecision(valid, "head"))
	valid.ReviewRequired = &required
	valid.Rationale = ""
	require.Error(t, ValidateDecision(valid, "head"))
}

func TestAssessmentCannotBeEmptyOrForAnotherRevision(t *testing.T) {
	score := 5.0
	report := mergeconfidence.ReportData{Complete: true, ReviewedCommit: "head", OverallScore: &score, OverallLevel: "positive", Checks: []mergeconfidence.ReportCheck{{ID: "security", Score: 5, MaxScore: 5, Summary: "Follows repository security practices."}}}
	require.True(t, ValidAssessment(report, "head"))
	require.False(t, ValidAssessment(report, "next"))
	report.Complete = false
	require.False(t, ValidAssessment(report, "head"))
	report.Complete = true
	report.Checks = nil
	require.False(t, ValidAssessment(report, "head"))
}
