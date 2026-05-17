package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AppSyncStatusOk      = "ok"
	AppSyncStatusSyncing = "syncing"
	AppSyncStatusFailed  = "failed"
)

type App struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID       uuid.UUID  `gorm:"type:uuid;not null"`
	DisplayName          string     `gorm:"not null"`
	Slug                 string     `gorm:"not null;uniqueIndex"`
	Description          string     `gorm:"not null;default:''"`
	CanvasID             *uuid.UUID `gorm:"type:uuid"`
	CodeStorageRepoID    string     `gorm:"not null;default:''"`
	CodeStorageRemoteURL string     `gorm:"not null;default:''"`
	DefaultBranch        string     `gorm:"not null;default:'main'"`
	LiveCommitSha        string     `gorm:"not null;default:''"`
	EditSessionBranch    *string
	SyncStatus           string  `gorm:"not null;default:'ok'"`
	SyncError            *string
	CreatedBy            *uuid.UUID `gorm:"type:uuid"`
	CreatedAt            *time.Time
	UpdatedAt            *time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (App) TableName() string {
	return "apps"
}

type AppDoc struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AppID     uuid.UUID `gorm:"type:uuid;not null"`
	Path      string    `gorm:"not null"`
	Content   string    `gorm:"not null;default:''"`
	Sha       string    `gorm:"not null;default:''"`
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func (AppDoc) TableName() string {
	return "app_docs"
}

// FindApp looks up an App by organization and app ID.
func FindApp(organizationID, appID uuid.UUID) (*App, error) {
	return FindAppInTransaction(database.Conn(), organizationID, appID)
}

func FindAppInTransaction(tx *gorm.DB, organizationID, appID uuid.UUID) (*App, error) {
	var app App
	err := tx.
		Where("id = ? AND organization_id = ?", appID, organizationID).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// FindAppBySlug looks up an App by organization and slug.
func FindAppBySlug(organizationID uuid.UUID, slug string) (*App, error) {
	var app App
	err := database.Conn().
		Where("organization_id = ? AND slug = ?", organizationID, slug).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// ListApps returns all non-deleted apps for an organization.
func ListApps(organizationID string) ([]App, error) {
	var apps []App
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Order("created_at ASC").
		Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}

// CreateApp creates a new App record.
func CreateApp(tx *gorm.DB, app *App) error {
	return tx.Clauses(clause.Returning{}).Create(app).Error
}

// UpdateApp saves changes to an existing App record.
func UpdateApp(tx *gorm.DB, app *App) error {
	now := time.Now()
	app.UpdatedAt = &now
	return tx.Save(app).Error
}

// SoftDeleteApp soft-deletes an App by setting deleted_at.
func (a *App) SoftDelete() error {
	return database.Conn().Delete(a).Error
}

// FindAppDocsByAppID returns all docs for an app.
func FindAppDocsByAppID(appID uuid.UUID) ([]AppDoc, error) {
	var docs []AppDoc
	err := database.Conn().
		Where("app_id = ?", appID).
		Order("path ASC").
		Find(&docs).Error
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// FindAppDocByPath returns a single doc by path.
func FindAppDocByPath(appID uuid.UUID, path string) (*AppDoc, error) {
	var doc AppDoc
	err := database.Conn().
		Where("app_id = ? AND path = ?", appID, path).
		First(&doc).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpsertAppDoc creates or updates a doc for an app.
func UpsertAppDoc(tx *gorm.DB, doc *AppDoc) (*AppDoc, error) {
	now := time.Now()
	doc.UpdatedAt = &now

	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "app_id"}, {Name: "path"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "sha", "updated_at"}),
	}).Create(doc).Error
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// IsAppSlugTaken returns true if the slug is already taken (globally unique).
func IsAppSlugTaken(slug string) (bool, error) {
	var count int64
	err := database.Conn().Model(&App{}).
		Where("slug = ?", slug).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ErrAppSlugAlreadyExists is returned when an app slug is already taken.
var ErrAppSlugAlreadyExists = errors.New("app slug already exists")
