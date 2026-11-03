package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MergeConfidencePublicationClaim serializes provider writes without keeping a
// database connection or transaction open while waiting for the provider.
type MergeConfidencePublicationClaim struct {
	PullRequestID uuid.UUID `gorm:"primaryKey"`
	OwnerID       uuid.UUID
	LeaseUntil    time.Time
}

func (c *MergeConfidencePublicationClaim) TableName() string {
	return "factory_merge_confidence_publication_claims"
}

func (c *MergeConfidencePublicationClaim) Acquire(tx *gorm.DB) (bool, error) {
	result := tx.Exec(`INSERT INTO factory_merge_confidence_publication_claims (pull_request_id, owner_id, lease_until)
		VALUES (?, ?, ?) ON CONFLICT (pull_request_id) DO UPDATE
		SET owner_id = EXCLUDED.owner_id, lease_until = EXCLUDED.lease_until
		WHERE factory_merge_confidence_publication_claims.lease_until <= clock_timestamp()`, c.PullRequestID, c.OwnerID, c.LeaseUntil)
	return result.RowsAffected == 1, result.Error
}

func (c *MergeConfidencePublicationClaim) IsCurrent(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Model(c).Where("pull_request_id = ? AND owner_id = ? AND lease_until > clock_timestamp()", c.PullRequestID, c.OwnerID).Count(&count).Error
	return count == 1, err
}

func (c *MergeConfidencePublicationClaim) Release(tx *gorm.DB) error {
	return tx.Where("pull_request_id = ? AND owner_id = ?", c.PullRequestID, c.OwnerID).Delete(c).Error
}
