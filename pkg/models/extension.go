package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/extensions"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ExtensionVersionStateDraft     = "draft"
	ExtensionVersionStatePublished = "published"
)

type Extension struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func CreateExtension(organizationID uuid.UUID, name string, description string) (*Extension, error) {
	now := time.Now()
	extension := &Extension{
		OrganizationID: organizationID,
		Name:           name,
		Description:    description,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := database.Conn().Create(extension).Error
	if err != nil {
		return nil, err
	}

	return extension, nil
}

func ListExtensions(organizationID uuid.UUID) ([]Extension, error) {
	extensions := []Extension{}
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Find(&extensions).
		Error

	if err != nil {
		return nil, err
	}

	return extensions, nil
}

func FindExtension(organizationID uuid.UUID, id string) (*Extension, error) {
	extension := &Extension{}
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("id = ?", id).
		First(&extension).
		Error

	if err != nil {
		return nil, err
	}

	return extension, nil
}

func FindExtensionByName(organizationID uuid.UUID, name string) (*Extension, error) {
	extension := &Extension{}
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("name = ?", name).
		First(&extension).
		Error

	if err != nil {
		return nil, err
	}

	return extension, nil
}

func (e *Extension) CreateVersionInTransaction(tx *gorm.DB, name string, digest string, manifest *extensions.Manifest) (*ExtensionVersion, error) {
	now := time.Now()
	version := &ExtensionVersion{
		OrganizationID: e.OrganizationID,
		ExtensionID:    e.ID,
		Name:           name,
		Digest:         digest,
		State:          ExtensionVersionStateDraft,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Manifest:       datatypes.NewJSONType(*manifest),
	}

	err := tx.Create(version).Error
	if err != nil {
		return nil, err
	}

	return version, nil
}

func (e *Extension) ListVersions() ([]ExtensionVersion, error) {
	versions := []ExtensionVersion{}
	err := database.Conn().
		Where("extension_id = ?", e.ID).
		Where("organization_id = ?", e.OrganizationID).
		Order("created_at DESC").
		Find(&versions).
		Error

	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (e *Extension) FindVersion(versionName string) (*ExtensionVersion, error) {
	version := &ExtensionVersion{}
	err := database.Conn().
		Where("organization_id = ?", e.OrganizationID).
		Where("extension_id = ?", e.ID).
		Where("name = ?", versionName).
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return version, nil
}

func (e *Extension) FindLatestVersion() (*ExtensionVersion, error) {
	version := &ExtensionVersion{}
	err := database.Conn().
		Where("organization_id = ?", e.OrganizationID).
		Where("extension_id = ?", e.ID).
		Order("created_at DESC").
		First(&version).
		Error

	if err != nil {
		return nil, err
	}

	return version, nil
}

type ExtensionVersion struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	ExtensionID    uuid.UUID
	Name           string
	Digest         string
	State          string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	PublishedAt    *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Manifest       datatypes.JSONType[extensions.Manifest]
}

func (v *ExtensionVersion) UpdateInTransaction(tx *gorm.DB, digest string, manifest *extensions.Manifest) error {
	now := time.Now()
	v.Digest = digest
	v.UpdatedAt = &now
	v.Manifest = datatypes.NewJSONType(*manifest)
	return tx.Save(v).Error
}

func (v *ExtensionVersion) Publish() error {
	now := time.Now()
	v.State = ExtensionVersionStatePublished
	v.PublishedAt = &now
	v.UpdatedAt = &now
	return database.Conn().Save(v).Error
}

func LoadManifest(organizationID string) (*extensions.Manifest, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	extensionList, err := ListExtensions(orgID)
	if err != nil {
		return nil, fmt.Errorf("error listing extensions: %w", err)
	}

	manifest := &extensions.Manifest{
		Integrations: []extensions.IntegrationManifest{},
		Components:   []extensions.ComponentManifest{},
		Triggers:     []extensions.TriggerManifest{},
	}

	for _, extension := range extensionList {
		version, err := extension.FindLatestVersion()
		if err != nil {
			return nil, fmt.Errorf("error finding latest version: %w", err)
		}

		manifest.Integrations = append(manifest.Integrations, version.Manifest.Data().Integrations...)
		manifest.Components = append(manifest.Components, version.Manifest.Data().Components...)
		manifest.Triggers = append(manifest.Triggers, version.Manifest.Data().Triggers...)
	}

	return manifest, nil
}
