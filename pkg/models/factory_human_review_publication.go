package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrHumanReviewPublicationBusy = errors.New("human review publication is in progress")

// WithHumanReviewPublicationLease reserves provider publication without holding
// a connection during network calls. The caller must supply a bounded context.
func (p *FactoryPullRequest) WithHumanReviewPublicationLease(db *gorm.DB, publish func(*gorm.DB) error) (err error) {
	ctx := db.Statement.Context
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("human review publication requires a deadline")
	}
	// Allow cancelled requests time to settle at GitHub before replacement.
	token, err := p.ClaimHumanReviewPublication(db, time.Until(deadline)+30*time.Second)
	if err != nil {
		return err
	}
	if token == uuid.Nil {
		return ErrHumanReviewPublicationBusy
	}
	defer func() {
		if ctx.Err() != nil {
			err = errors.Join(err, ctx.Err())
			return
		}
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err = errors.Join(err, p.ReleaseHumanReviewPublication(db.WithContext(releaseCtx), token))
	}()
	return publish(db)
}

// ClaimHumanReviewPublication uses a short transaction to coordinate with policy
// saves and run admission. No database lock remains held during publication.
func (p *FactoryPullRequest) ClaimHumanReviewPublication(db *gorm.DB, duration time.Duration) (uuid.UUID, error) {
	token := uuid.New()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := p.lockHumanReviewPublication(tx); err != nil {
			return err
		}
		result := tx.Exec(`
			INSERT INTO factory_human_review_publication_leases (pull_request_id, token, expires_at)
			VALUES (?, ?, clock_timestamp() + ? * interval '1 millisecond')
			ON CONFLICT (pull_request_id) DO UPDATE
			SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at
			WHERE factory_human_review_publication_leases.expires_at <= clock_timestamp()
		`, p.ID, token, duration.Milliseconds())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrHumanReviewPublicationBusy
		}
		return nil
	})
	if errors.Is(err, ErrHumanReviewPublicationBusy) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	return token, nil
}

// RequireHumanReviewPublicationIdle prevents policy and authority changes during
// publication. Call inside the transaction that makes the change; retry if busy.
func (p *FactoryPullRequest) RequireHumanReviewPublicationIdle(tx *gorm.DB) error {
	if err := p.lockHumanReviewPublication(tx); err != nil {
		return err
	}
	var busy bool
	if err := tx.Raw(`SELECT EXISTS (
		SELECT 1 FROM factory_human_review_publication_leases
		WHERE pull_request_id = ? AND expires_at > clock_timestamp()
	)`, p.ID).Scan(&busy).Error; err != nil {
		return err
	}
	if busy {
		return ErrHumanReviewPublicationBusy
	}
	return nil
}

// ReleaseHumanReviewPublication cannot release a replacement worker's claim.
func (p *FactoryPullRequest) ReleaseHumanReviewPublication(db *gorm.DB, token uuid.UUID) error {
	return db.Exec(`DELETE FROM factory_human_review_publication_leases WHERE pull_request_id = ? AND token = ?`, p.ID, token).Error
}

func (p *FactoryPullRequest) lockHumanReviewPublication(tx *gorm.DB) error {
	var acquired bool
	if err := tx.Raw("SELECT pg_try_advisory_xact_lock(hashtext(?))", "human-review:"+p.ID.String()).Scan(&acquired).Error; err != nil {
		return err
	}
	if !acquired {
		return ErrHumanReviewPublicationBusy
	}
	return nil
}

// RecordPublication updates only delivery fields. A concurrent agent completion
// retains its immutable Result and decision state.
func (d *FactoryHumanReviewDecision) RecordPublication(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return tx.Model(d).Where("state <> ?", FactoryHumanReviewStateSuperseded).Updates(map[string]any{
		"check_run_id":      d.CheckRunID,
		"check_url":         d.CheckURL,
		"publication_error": d.PublicationError,
		"blocking_reason":   d.BlockingReason,
		"updated_at":        d.UpdatedAt,
	}).Error
}
