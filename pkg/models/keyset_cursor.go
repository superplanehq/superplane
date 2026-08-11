package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KeysetCursor addresses a position in a listing ordered by
// (created_at DESC, id DESC).
//
// The id half is what lets the cursor land inside a group of rows sharing a
// created_at. With a timestamp alone, a page that ends inside such a group
// leaves a cursor of that timestamp, and "created_at < cursor" then excludes
// every remaining row of the group - those rows are not duplicated, they are
// dropped. Equal timestamps are ordinary: batch inserts share one now(), and
// Postgres stores microsecond precision.
//
// ID is optional so that clients sending only a timestamp keep the behavior
// they have today.
type KeysetCursor struct {
	CreatedAt *time.Time
	ID        uuid.UUID
}

// Apply adds the cursor's WHERE clause to query. The caller is responsible for
// ordering by created_at DESC, id DESC - the comparison only pages correctly
// against that order.
func (c *KeysetCursor) Apply(query *gorm.DB) *gorm.DB {
	if c == nil || c.CreatedAt == nil {
		return query
	}

	if c.ID == uuid.Nil {
		return query.Where("created_at < ?", c.CreatedAt)
	}

	return query.Where("(created_at, id) < (?, ?)", c.CreatedAt, c.ID)
}
