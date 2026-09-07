package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaxUserLastLocationPathBytes bounds the stored path so a malformed or
// abusive client cannot grow the row without limit. Deep links in the app
// are short (organization slug, resource id, a few query params), so this
// leaves generous headroom.
const MaxUserLastLocationPathBytes = 2048

var ErrUserLastLocationPathInvalid = errors.New("path must be a relative in-app path")

// UserLastLocation is the last in-app screen a user visited, scoped to the
// organization they were in at the time. It powers "resume where you left
// off" on login: if the user closes the browser (or opens a new one) while
// waiting on something like a pending approval, they land back on that exact
// screen instead of the organization home page.
type UserLastLocation struct {
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	UserID         uuid.UUID `gorm:"primaryKey"`
	Path           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (l *UserLastLocation) TableName() string {
	return "user_last_locations"
}

// IsValidUserLastLocationPath reports whether path is safe to store and
// later redirect a user to. It must be a same-origin, relative path (so we
// never redirect off SuperPlane) and stay within the size limit.
func IsValidUserLastLocationPath(path string) bool {
	if path == "" || len(path) > MaxUserLastLocationPathBytes {
		return false
	}

	// A leading "/" makes it relative to the app origin. A second leading
	// "/" (or "\") turns it into a protocol-relative URL in a browser
	// (e.g. "//evil.com"), so reject those too.
	if path[0] != '/' || len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		return false
	}

	return true
}

func FindUserLastLocation(tx *gorm.DB, organizationID, userID uuid.UUID) (*UserLastLocation, error) {
	var location UserLastLocation
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		First(&location).
		Error
	if err != nil {
		return nil, err
	}

	return &location, nil
}

// SetUserLastLocation upserts the calling user's last known location for
// the organization. It returns ErrUserLastLocationPathInvalid if path is
// not a safe relative in-app path.
func SetUserLastLocation(tx *gorm.DB, organizationID, userID uuid.UUID, path string) (*UserLastLocation, error) {
	if !IsValidUserLastLocationPath(path) {
		return nil, ErrUserLastLocationPathInvalid
	}

	now := time.Now()
	location := &UserLastLocation{
		OrganizationID: organizationID,
		UserID:         userID,
		Path:           path,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := tx.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"path", "updated_at"}),
		}).
		Create(location).
		Error
	if err != nil {
		return nil, err
	}

	return FindUserLastLocation(tx, organizationID, userID)
}
