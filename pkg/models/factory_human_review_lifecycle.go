package models

import (
	"fmt"

	"gorm.io/gorm"
)

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
