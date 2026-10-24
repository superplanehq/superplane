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

// Reserve at most two connections for renewal schedules in each process.
// The default pool has five connections; queued schedules wait here instead
// of occupying connections that foreground requests need. API schedules and
// reconciliation workers share this limit, including advisory-lock waiters.
var seatRenewalScheduleSlots = make(chan struct{}, 2)

// WithSeatRenewalScheduleLock serializes renewal schedules across processes.
// The session lock spans HTTP calls without holding a transaction or row lock.
// Admission is bounded before reserving a connection. Local transactions use
// that reserved connection, so the lock holder does not need another slot.
func WithSeatRenewalScheduleLock(db *gorm.DB, orgID uuid.UUID, schedule func(*gorm.DB) error) error {
	select {
	case seatRenewalScheduleSlots <- struct{}{}:
		defer func() { <-seatRenewalScheduleSlots }()
	case <-db.Statement.Context.Done():
		return fmt.Errorf("wait for seat renewal scheduling capacity: %w", db.Statement.Context.Err())
	}

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
