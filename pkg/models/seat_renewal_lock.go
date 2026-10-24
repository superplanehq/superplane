package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WithSeatRenewalScheduleLock serializes renewal schedules across processes.
// The session lock spans HTTP calls without holding a transaction or row lock.
// Local transactions use the same reserved connection, so waiting schedulers
// cannot exhaust the pool and prevent the lock holder from finishing.
func WithSeatRenewalScheduleLock(db *gorm.DB, orgID uuid.UUID, schedule func(*gorm.DB) error) error {
	key := "seat-renewal:" + orgID.String()
	return db.Connection(func(conn *gorm.DB) (err error) {
		if err := conn.Exec("SELECT pg_advisory_lock(hashtextextended(?, 0))", key).Error; err != nil {
			return fmt.Errorf("lock seat renewal schedule: %w", err)
		}
		defer func() {
			// A canceled schedule must still release its session lock before
			// the connection returns to the pool.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if unlockErr := conn.WithContext(ctx).Exec("SELECT pg_advisory_unlock(hashtextextended(?, 0))", key).Error; unlockErr != nil {
				// Discard a connection whose lock could not be released.
				if sqlConn, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = sqlConn.Raw(func(any) error { return driver.ErrBadConn })
				}
				err = errors.Join(err, fmt.Errorf("unlock seat renewal schedule: %w", unlockErr))
			}
		}()
		return schedule(conn)
	})
}
