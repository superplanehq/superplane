package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequestHumanReviewReconciliation keeps reopen events recoverable even when
// the bounded queue rejects them or the process exits before reconciliation.
func (p *FactoryPullRequest) RequestHumanReviewReconciliation(tx *gorm.DB) error {
	return tx.Model(p).Update("human_review_request_id", uuid.New()).Error
}

// CompleteHumanReviewReconciliation must not erase a newer webhook request
// received while the provider read or check publication was in flight.
func (p *FactoryPullRequest) CompleteHumanReviewReconciliation(tx *gorm.DB, requestID *uuid.UUID) error {
	if requestID == nil {
		return nil
	}
	return tx.Model(p).Where("human_review_request_id = ?", *requestID).Update("human_review_request_id", nil).Error
}

// ReopenForHumanReview is called only after a current GitHub read confirms
// that the tracked PR is open. Historical close timestamps must no longer
// classify the PR as closed after that transition.
func (p *FactoryPullRequest) ReopenForHumanReview(tx *gorm.DB, state string) error {
	if state != FactoryPullRequestStateOpen && state != FactoryPullRequestStateDraft {
		return fmt.Errorf("invalid open pull request state %q", state)
	}
	if err := p.Update(tx, FactoryPullRequestPatch{State: &state}); err != nil {
		return err
	}
	p.ClosedAt = nil
	return tx.Model(p).Update("closed_at", nil).Error
}
