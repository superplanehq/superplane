package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	LicenseDeactivationReasonExpired  = "expired"
	LicenseDeactivationReasonRevoked  = "revoked"
	LicenseDeactivationReasonRemoved  = "removed"
	LicenseDeactivationReasonReplaced = "replaced"
)

// RemovedEnterpriseGrant is one grant captured when a lifecycle event deletes it.
type RemovedEnterpriseGrant struct {
	OrganizationID uuid.UUID  `json:"organization_id"`
	FeatureKey     string     `json:"feature_key"`
	Source         string     `json:"source"`
	GrantedBy      *uuid.UUID `json:"granted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	RemovedAt      time.Time  `json:"removed_at"`
	RemovedBy      string     `json:"removed_by"`
}

// InstallationLicenseActivation is one episode of an active installation license.
// At most one row has deactivated_at NULL.
type InstallationLicenseActivation struct {
	ID                 int64
	LicenseID          uuid.UUID
	LicenseFeatures    pq.StringArray `gorm:"type:text[]"`
	ActivatedAt        time.Time
	OrganizationCount  int
	AutoGranted        bool
	DeactivatedAt      *time.Time
	DeactivationReason *string
	RemovedGrants      datatypes.JSONSlice[RemovedEnterpriseGrant]
}

func (InstallationLicenseActivation) TableName() string {
	return "installation_license_activations"
}

// FindLicenseActivationByID returns nil, nil when the row does not exist.
func FindLicenseActivationByID(tx *gorm.DB, id int64) (*InstallationLicenseActivation, error) {
	var row InstallationLicenseActivation
	err := tx.First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &row, nil
}

// FindOpenLicenseActivation returns nil, nil when no activation is open.
func FindOpenLicenseActivation(tx *gorm.DB) (*InstallationLicenseActivation, error) {
	return takeOpenLicenseActivation(tx, false)
}

// LockOpenLicenseActivationForShare locks the open row so a grant write
// sees the same license features as a lifecycle event.
func LockOpenLicenseActivationForShare(tx *gorm.DB) (*InstallationLicenseActivation, error) {
	return takeOpenLicenseActivation(tx, true)
}

// InsertOpenLicenseActivation inserts an open activation. inserted is false
// when another open row already exists. The partial unique index makes the
// insert run once across replicas.
func InsertOpenLicenseActivation(tx *gorm.DB, activation *InstallationLicenseActivation) (bool, error) {
	features := []string(activation.LicenseFeatures)
	if features == nil {
		features = []string{}
	}

	row := tx.Raw(`
		INSERT INTO installation_license_activations (
			license_id,
			license_features,
			organization_count,
			auto_granted
		)
		VALUES (?, ?::text[], ?, ?)
		ON CONFLICT ((true)) WHERE deactivated_at IS NULL DO NOTHING
		RETURNING id, activated_at
	`, activation.LicenseID, pq.Array(features), activation.OrganizationCount, activation.AutoGranted).Row()

	var id int64
	var activatedAt time.Time
	err := row.Scan(&id, &activatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	activation.ID = id
	activation.ActivatedAt = activatedAt
	activation.LicenseFeatures = features
	if activation.RemovedGrants == nil {
		activation.RemovedGrants = datatypes.JSONSlice[RemovedEnterpriseGrant]{}
	}

	return true, nil
}

// CloseLicenseActivation closes an open row. closed is false when the row
// was already closed.
func CloseLicenseActivation(tx *gorm.DB, id int64, reason string) (bool, error) {
	result := tx.Exec(`
		UPDATE installation_license_activations
		SET deactivated_at = now(), deactivation_reason = ?
		WHERE id = ? AND deactivated_at IS NULL
	`, reason, id)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// SetLicenseActivationResult stores the organization count and whether this
// activation inserted automatic grants.
func SetLicenseActivationResult(tx *gorm.DB, id int64, organizationCount int, autoGranted bool) error {
	return tx.Model(&InstallationLicenseActivation{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"organization_count": organizationCount,
			"auto_granted":       autoGranted,
		}).Error
}

func AppendRemovedGrants(tx *gorm.DB, id int64, removed []RemovedEnterpriseGrant) error {
	if len(removed) == 0 {
		return nil
	}

	payload, err := json.Marshal(removed)
	if err != nil {
		return err
	}

	return tx.Exec(`
		UPDATE installation_license_activations
		SET removed_grants = removed_grants || ?::jsonb
		WHERE id = ?
	`, payload, id).Error
}

func ListLicenseActivations(tx *gorm.DB, limit, offset int) ([]InstallationLicenseActivation, int64, error) {
	var total int64
	if err := tx.Model(&InstallationLicenseActivation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := tx.Order("activated_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var rows []InstallationLicenseActivation
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// CountOrganizations counts organizations that are not deleted. Suspended
// organizations are included.
func CountOrganizations(tx *gorm.DB) (int64, error) {
	var count int64
	err := tx.Model(&Organization{}).Count(&count).Error
	return count, err
}

// FindOnlyOrganization returns the organization when exactly one exists.
// It returns nil, nil otherwise.
func FindOnlyOrganization(tx *gorm.DB) (*Organization, error) {
	var organizations []Organization
	err := tx.Limit(2).Find(&organizations).Error
	if err != nil {
		return nil, err
	}
	if len(organizations) != 1 {
		return nil, nil
	}

	return &organizations[0], nil
}

func takeOpenLicenseActivation(tx *gorm.DB, share bool) (*InstallationLicenseActivation, error) {
	query := tx
	if share {
		query = tx.Clauses(clause.Locking{Strength: "SHARE"})
	}

	var row InstallationLicenseActivation
	err := query.Where("deactivated_at IS NULL").Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &row, nil
}
