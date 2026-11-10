// Package humanreview owns server-enforced review policy independently of agent instructions.
package humanreview

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
)

const CheckName = "merge-confidence/merge-gate"

type Reviewer struct {
	ID     int64  `json:"id"`
	Login  string `json:"login"`
	Reason string `json:"reason"`
}

type Decision struct {
	AssessedCommit    string     `json:"assessedCommit"`
	ReviewRequired    *bool      `json:"reviewRequired"`
	Rationale         string     `json:"rationale"`
	SelectedReviewers []Reviewer `json:"selectedReviewers"`
}

type Review struct {
	ID          int64
	UserID      int64
	State       string
	CommitID    string
	SubmittedAt time.Time
}

type GateInput struct {
	Enabled         bool
	AssessmentValid bool
	DecisionValid   bool
	HeadSHA         string
	Author          string
	Decision        Decision
	Eligible        map[int64]bool
	Reviews         []Review
}

type Gate struct {
	Status      string
	Conclusion  string
	Reason      string
	ApprovalIDs []int64
}

func ValidateDecision(decision Decision, expectedCommit string) error {
	if strings.TrimSpace(expectedCommit) == "" || decision.AssessedCommit != expectedCommit {
		return fmt.Errorf("decision does not cover the current assessed commit")
	}
	if decision.ReviewRequired == nil || strings.TrimSpace(decision.Rationale) == "" {
		return fmt.Errorf("decision must include reviewRequired and a rationale")
	}
	if len(decision.Rationale) > 16000 || len(decision.SelectedReviewers) > 100 {
		return fmt.Errorf("decision exceeds the result limit")
	}
	seen := map[int64]bool{}
	for _, reviewer := range decision.SelectedReviewers {
		if reviewer.ID <= 0 || strings.TrimSpace(reviewer.Login) == "" || strings.TrimSpace(reviewer.Reason) == "" || seen[reviewer.ID] {
			return fmt.Errorf("each selected reviewer must have a unique ID, login, and selection reason")
		}
		seen[reviewer.ID] = true
	}
	return nil
}

func ValidAssessment(report mergeconfidence.ReportData, headSHA string) bool {
	if !report.Complete || report.ReviewedCommit != headSHA || headSHA == "" || len(report.Checks) == 0 || report.OverallScore == nil {
		return false
	}
	if math.IsNaN(*report.OverallScore) || *report.OverallScore < 1 || *report.OverallScore > 5 || strings.TrimSpace(report.OverallLevel) == "" {
		return false
	}
	for _, check := range report.Checks {
		if check.Missing || check.MaxScore != 5 || check.Score < 1 || check.Score > 5 || math.Trunc(check.Score) != check.Score || strings.TrimSpace(check.Summary) == "" {
			return false
		}
	}
	return true
}

func EvaluateGate(input GateInput) Gate {
	if !input.Enabled {
		return Gate{Status: "completed", Conclusion: "success", Reason: "Request review enforcement is disabled."}
	}
	if !input.AssessmentValid {
		return blocked("Merge confidence failed or has an incomplete assessment. Rerun merge confidence.")
	}
	if !input.DecisionValid || ValidateDecision(input.Decision, input.HeadSHA) != nil {
		return blocked("The human review decision is missing, invalid, or superseded. Rerun Request review.")
	}
	latest := latestDecisiveReviews(input.Reviews)
	for userID, review := range latest {
		if input.Eligible[userID] && strings.EqualFold(review.State, "CHANGES_REQUESTED") {
			return blocked("An eligible reviewer requested changes. Obtain their approval or dismiss the review in GitHub.")
		}
	}
	if !*input.Decision.ReviewRequired {
		return Gate{Status: "completed", Conclusion: "success", Reason: input.Decision.Rationale}
	}
	gate := blocked("A selected reviewer must approve the current head commit.")
	for _, selected := range input.Decision.SelectedReviewers {
		if !input.Eligible[selected.ID] || strings.EqualFold(selected.Login, input.Author) {
			continue
		}
		review, ok := latest[selected.ID]
		if ok && strings.EqualFold(review.State, "APPROVED") && review.CommitID == input.HeadSHA {
			gate.ApprovalIDs = append(gate.ApprovalIDs, review.ID)
		}
	}
	if len(gate.ApprovalIDs) > 0 {
		gate.Status, gate.Conclusion, gate.Reason = "completed", "success", "A selected reviewer approved the current head commit."
	}
	return gate
}

// Comments do not change review disposition. GitHub returns dismissed reviews
// with state DISMISSED, so they revoke only that review, not an older request
// for changes from the same reviewer that remains unresolved.
func latestDecisiveReviews(reviews []Review) map[int64]Review {
	latest := map[int64]Review{}
	for _, review := range reviews {
		state := strings.ToUpper(review.State)
		if state != "APPROVED" && state != "CHANGES_REQUESTED" {
			continue
		}
		previous, exists := latest[review.UserID]
		if !exists || review.SubmittedAt.After(previous.SubmittedAt) || (review.SubmittedAt.Equal(previous.SubmittedAt) && review.ID > previous.ID) {
			latest[review.UserID] = review
		}
	}
	return latest
}

func blocked(reason string) Gate {
	return Gate{Status: "completed", Conclusion: "failure", Reason: reason}
}
