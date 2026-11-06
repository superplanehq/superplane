package models

import (
	"context"
	"database/sql/driver"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// WithHumanReviewPublicationLock serializes provider mutations on a dedicated
// connection without holding a database transaction open during network calls.
func (p *FactoryPullRequest) WithHumanReviewPublicationLock(db *gorm.DB, publish func(*gorm.DB) error) error {
	return db.Connection(func(conn *gorm.DB) error {
		// Reuse the connection without carrying SQL clauses between queries.
		conn = conn.Session(&gorm.Session{NewDB: true})
		key := "human-review:" + p.ID.String()
		if err := conn.Exec("SELECT pg_advisory_lock(hashtext(?))", key).Error; err != nil {
			return err
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(db.Statement.Context), 5*time.Second)
			defer cancel()
			if err := conn.WithContext(releaseCtx).Exec("SELECT pg_advisory_unlock(hashtext(?))", key).Error; err != nil {
				log.WithError(err).Error("Human review: failed to release publication lock")
				if connection, ok := conn.Statement.ConnPool.(interface{ Raw(func(any) error) error }); ok {
					_ = connection.Raw(func(any) error { return driver.ErrBadConn })
				}
			}
		}()
		return publish(conn)
	})
}

// RecordPublication updates only delivery fields. A concurrent agent completion
// retains its immutable Result and decision state.
func (d *FactoryHumanReviewDecision) RecordPublication(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return tx.Model(d).Where("state <> ?", FactoryHumanReviewStateSuperseded).Updates(map[string]any{
		"check_run_id":        d.CheckRunID,
		"check_url":           d.CheckURL,
		"publication_pending": d.PublicationPending,
		"publication_error":   d.PublicationError,
		"retry_at":            d.RetryAt,
		"blocking_reason":     d.BlockingReason,
		"updated_at":          d.UpdatedAt,
	}).Error
}
