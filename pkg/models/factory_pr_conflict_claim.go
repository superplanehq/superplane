package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FactoryPRConflictClaim records the pull request head that already started
// a conflict repair for one handler. A new head can start another repair.
type FactoryPRConflictClaim struct {
	HandlerID     uuid.UUID `gorm:"primaryKey"`
	PullRequestID uuid.UUID `gorm:"primaryKey"`
	HeadSHA       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (FactoryPRConflictClaim) TableName() string {
	return "factory_pr_conflict_claims"
}

// ClaimFactoryPullRequestConflictHead inserts the head, or replaces a different
// head, for one handler and pull request. The same head returns false.
func ClaimFactoryPullRequestConflictHead(tx *gorm.DB, handlerID, pullRequestID uuid.UUID, headSHA string) (bool, error) {
	headSHA = strings.TrimSpace(headSHA)
	if handlerID == uuid.Nil || pullRequestID == uuid.Nil || headSHA == "" {
		return false, nil
	}

	now := time.Now()
	result := tx.Exec(`
INSERT INTO factory_pr_conflict_claims (handler_id, pull_request_id, head_sha, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (handler_id, pull_request_id) DO UPDATE
SET head_sha = EXCLUDED.head_sha, updated_at = EXCLUDED.updated_at
WHERE factory_pr_conflict_claims.head_sha <> EXCLUDED.head_sha
`, handlerID, pullRequestID, headSHA, now, now)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// ReleaseFactoryPullRequestConflictHead drops a claim that did not emit an event.
func ReleaseFactoryPullRequestConflictHead(tx *gorm.DB, handlerID, pullRequestID uuid.UUID, headSHA string) error {
	headSHA = strings.TrimSpace(headSHA)
	if handlerID == uuid.Nil || pullRequestID == uuid.Nil || headSHA == "" {
		return nil
	}
	return tx.Exec(`
DELETE FROM factory_pr_conflict_claims
WHERE handler_id = ? AND pull_request_id = ? AND head_sha = ?
`, handlerID, pullRequestID, headSHA).Error
}
