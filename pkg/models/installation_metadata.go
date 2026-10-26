package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const installationMetadataID = 1

const (
	OwnerSetupStepLicense       = "license"
	OwnerSetupStepSMTP          = "smtp"
	OwnerSetupStepFleetIntro    = "fleet-intro"
	OwnerSetupStepFleetProvider = "fleet-provider"
	OwnerSetupStepFleetFleets   = "fleet-fleets"
	OwnerSetupStepFleetValues   = "fleet-values"
	OwnerSetupStepFleetInstall  = "fleet-install"
	OwnerSetupStepFleetDocker   = "fleet-docker"
)

// InstallationMetadata holds general installation settings.
type InstallationMetadata struct {
	ID                        int    `gorm:"primary_key"`
	InstallationID            string `gorm:"type:varchar(64)"`
	AllowPrivateNetworkAccess bool
	SignupsEnabled            bool
	InvitationDurationDays    int     `gorm:"default:7"`
	OwnerSetupStep            *string `gorm:"type:text"`
	ShareOwnerIdentity        bool
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// OwnerSetupSteps lists the wizard steps an installation admin can resume.
// The survey step is not persisted: reaching it ends the wizard.
var OwnerSetupSteps = []string{
	OwnerSetupStepLicense,
	OwnerSetupStepSMTP,
	OwnerSetupStepFleetIntro,
	OwnerSetupStepFleetProvider,
	OwnerSetupStepFleetFleets,
	OwnerSetupStepFleetValues,
	OwnerSetupStepFleetInstall,
	OwnerSetupStepFleetDocker,
}

func IsOwnerSetupStep(step string) bool {
	return slices.Contains(OwnerSetupSteps, step)
}

func GetInstallationMetadata(tx *gorm.DB) (*InstallationMetadata, error) {
	return findOrCreateInstallationMetadata(tx)
}

func GetInstallationID(tx *gorm.DB) (string, error) {
	metadata, err := findOrCreateInstallationMetadata(tx)
	if err != nil {
		return "", err
	}

	return metadata.InstallationID, nil
}

// GetOwnerSetupStep returns the owner setup wizard step in progress.
// An empty value means the wizard is not in progress.
func GetOwnerSetupStep(tx *gorm.DB) (string, error) {
	metadata, err := findOrCreateInstallationMetadata(tx)
	if err != nil {
		return "", err
	}

	if metadata.OwnerSetupStep == nil {
		return "", nil
	}

	return *metadata.OwnerSetupStep, nil
}

// SetOwnerSetupStep records the current owner setup wizard step.
// An empty step clears the value and closes the wizard.
func SetOwnerSetupStep(tx *gorm.DB, step string) error {
	if _, err := findOrCreateInstallationMetadata(tx); err != nil {
		return err
	}

	var value any
	if step != "" {
		value = step
	}

	return tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Update("owner_setup_step", value).
		Error
}

// SetOwnerSetupStepIfOpen updates the wizard step only while it is open.
// A close and a later step write cannot reopen a finished wizard.
func SetOwnerSetupStepIfOpen(tx *gorm.DB, step string) (bool, error) {
	updated := false
	err := tx.Transaction(func(inner *gorm.DB) error {
		if _, err := findOrCreateInstallationMetadata(inner); err != nil {
			return err
		}

		var metadata InstallationMetadata
		err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", installationMetadataID).
			First(&metadata).Error
		if err != nil {
			return err
		}
		if metadata.OwnerSetupStep == nil || *metadata.OwnerSetupStep == "" {
			return nil
		}

		var value any = step
		if step == "" {
			value = gorm.Expr("NULL")
		}

		result := inner.Model(&InstallationMetadata{}).
			Where("id = ? AND owner_setup_step IS NOT NULL AND owner_setup_step <> ''", installationMetadataID).
			Update("owner_setup_step", value)
		if result.Error != nil {
			return result.Error
		}

		updated = result.RowsAffected > 0
		return nil
	})

	return updated, err
}

// SetShareOwnerIdentity records whether weekly install reports may include
// the owner email and the first organization name.
func SetShareOwnerIdentity(tx *gorm.DB, share bool) error {
	if _, err := findOrCreateInstallationMetadata(tx); err != nil {
		return err
	}

	return tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Updates(map[string]any{
			"share_owner_identity": share,
			"updated_at":           time.Now(),
		}).
		Error
}

func UpdateInstallationMetadata(tx *gorm.DB, metadata *InstallationMetadata) error {
	return writeInstallationMetadata(tx, metadata)
}

// ApplyInstallationMetadata updates installation settings while the row is locked.
// The callback sees the current values, so one save cannot restore fields from an older read.
func ApplyInstallationMetadata(tx *gorm.DB, apply func(*InstallationMetadata)) error {
	metadata, err := lockInstallationMetadata(tx)
	if err != nil {
		return err
	}

	apply(metadata)
	return writeInstallationMetadata(tx, metadata)
}

func writeInstallationMetadata(tx *gorm.DB, metadata *InstallationMetadata) error {
	if _, err := findOrCreateInstallationMetadata(tx); err != nil {
		return err
	}

	return tx.Model(&InstallationMetadata{}).
		Where("id = ?", installationMetadataID).
		Updates(map[string]any{
			"allow_private_network_access": metadata.AllowPrivateNetworkAccess,
			"signups_enabled":              metadata.SignupsEnabled,
			"invitation_duration_days":     metadata.InvitationDurationDays,
			"updated_at":                   metadata.UpdatedAt,
		}).
		Error
}

func lockInstallationMetadata(tx *gorm.DB) (*InstallationMetadata, error) {
	if _, err := findOrCreateInstallationMetadata(tx); err != nil {
		return nil, err
	}

	var metadata InstallationMetadata
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", installationMetadataID).
		First(&metadata).Error
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func findOrCreateInstallationMetadata(tx *gorm.DB) (*InstallationMetadata, error) {
	var metadata InstallationMetadata
	err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error
	if err == nil {
		return installationMetadataWithID(tx, &metadata)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	metadata = InstallationMetadata{
		ID:                     installationMetadataID,
		InstallationID:         uuid.NewString(),
		SignupsEnabled:         true,
		InvitationDurationDays: 7,
	}

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&metadata).Error; err != nil {
		return nil, err
	}

	if err := tx.Where("id = ?", installationMetadataID).First(&metadata).Error; err != nil {
		return nil, err
	}

	return installationMetadataWithID(tx, &metadata)
}

func installationMetadataWithID(tx *gorm.DB, metadata *InstallationMetadata) (*InstallationMetadata, error) {
	if validInstallationID(metadata.InstallationID) {
		return metadata, nil
	}

	next := uuid.NewString()
	result := tx.Model(&InstallationMetadata{}).
		Where("id = ? AND installation_id = ?", installationMetadataID, metadata.InstallationID).
		Update("installation_id", next)
	if result.Error != nil {
		return nil, result.Error
	}

	if err := tx.Where("id = ?", installationMetadataID).First(metadata).Error; err != nil {
		return nil, err
	}

	return metadata, nil
}

func validInstallationID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed != uuid.Nil
}
