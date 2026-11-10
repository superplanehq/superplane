package models

import (
	"context"
	"database/sql/driver"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// WithMergeConfidencePublicationLock serializes check delivery on a dedicated
// connection. Competing workers leave their requests pending for the next poll.
func (p *FactoryPullRequest) WithMergeConfidencePublicationLock(db *gorm.DB, publish func(*gorm.DB) error) error {
	return db.Connection(func(conn *gorm.DB) error {
		conn = conn.Session(&gorm.Session{NewDB: true})
		key := "merge-confidence:" + p.ID.String()
		discard := func() {
			if connection, ok := conn.Statement.ConnPool.(interface{ Raw(func(any) error) error }); ok {
				_ = connection.Raw(func(any) error { return driver.ErrBadConn })
			}
		}
		var acquired bool
		if err := conn.Raw("SELECT pg_try_advisory_lock(hashtext(?))", key).Scan(&acquired).Error; err != nil {
			// The server may have acquired the lock before its response was lost.
			discard()
			return err
		}
		if !acquired {
			return nil
		}
		defer func() {
			releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(db.Statement.Context), 5*time.Second)
			defer cancel()
			var released bool
			err := conn.WithContext(releaseCtx).Raw("SELECT pg_advisory_unlock(hashtext(?))", key).Scan(&released).Error
			if err == nil && !released {
				err = errors.New("merge confidence publication lock was not held")
			}
			if err != nil {
				log.WithError(err).Error("Merge confidence: failed to release publication lock")
				discard()
			}
		}()
		return publish(conn)
	})
}
