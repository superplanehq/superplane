package models

import (
	"errors"
	"strings"
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

// PathBelongsToOrganization reports whether path is an in-app URL for slug.
// Accepts "/{slug}", "/{slug}/...", "/{slug}?...", and "/{slug}#...".
func PathBelongsToOrganization(path, slug string) bool {
	if slug == "" || !IsValidUserLastLocationPath(path) {
		return false
	}

	prefix := "/" + slug
	if path == prefix {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	switch path[len(prefix)] {
	case '/', '?', '#':
		return true
	default:
		return false
	}
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

	organization, err := FindOrganizationByIDInTransaction(tx, organizationID.String())
	if err != nil {
		return nil, err
	}
	if !PathBelongsToOrganization(path, organization.Slug) {
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

	err = tx.
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

// ListUserLastLocationsForAccount returns saved resume paths for every
// active organization the account still belongs to.
func ListUserLastLocationsForAccount(tx *gorm.DB, accountID uuid.UUID) ([]UserLastLocation, error) {
	var locations []UserLastLocation
	err := tx.
		Model(&UserLastLocation{}).
		Joins("JOIN users ON users.id = user_last_locations.user_id AND users.deleted_at IS NULL").
		Joins("JOIN organizations ON organizations.id = user_last_locations.organization_id AND organizations.deleted_at IS NULL").
		Where("users.account_id = ?", accountID).
		Find(&locations).
		Error
	return locations, err
}
