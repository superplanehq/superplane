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

// Reserve at most two connections for seat-quantity mutations in each process.
// The default pool has five connections; queued mutations wait here instead
// of occupying connections that foreground requests need. Admin removals,
// purchases, and reconciliation share this limit, including advisory-lock waiters.
var seatQuantitySlots = make(chan struct{}, 2)

// WithSeatQuantityLock serializes seat-quantity mutations across processes.
// The session lock spans provider HTTP without holding a transaction or a row
// lock. Admission is bounded before reserving a connection. Local transactions
// can use another connection from the pool; the lock holder keeps this one.
func WithSeatQuantityLock(db *gorm.DB, orgID uuid.UUID, run func(*gorm.DB) error) error {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case seatQuantitySlots <- struct{}{}:
		defer func() { <-seatQuantitySlots }()
	case <-ctx.Done():
		return fmt.Errorf("wait for seat quantity capacity: %w", ctx.Err())
	}

	key := "seat-quantity:" + orgID.String()
	return db.Connection(func(conn *gorm.DB) (err error) {
		if err := conn.Exec("SELECT pg_advisory_lock(hashtextextended(?, 0))", key).Error; err != nil {
			return fmt.Errorf("lock seat quantity: %w", err)
		}
		defer func() {
			// A canceled mutation must still release its session lock before
			// the connection returns to the pool.
			unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if unlockErr := conn.WithContext(unlockCtx).Exec("SELECT pg_advisory_unlock(hashtextextended(?, 0))", key).Error; unlockErr != nil {
				if sqlConn, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = sqlConn.Raw(func(any) error { return driver.ErrBadConn })
				}
				err = errors.Join(err, fmt.Errorf("unlock seat quantity: %w", unlockErr))
			}
		}()
		return run(conn)
	})
}
