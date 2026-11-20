package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AutomationSource stores the authored definition and the artifact published for its revision.
type AutomationSource struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	FactoryID        uuid.UUID
	Name             string
	Source           string
	Revision         int64
	ArtifactID       string
	ArtifactRevision string
}

func (a *AutomationSource) TableName() string {
	return "automations_as_code"
}

func (a *AutomationSource) Save(tx *gorm.DB) error {
	return tx.Save(a).Error
}

func (a *AutomationSource) Delete(tx *gorm.DB) error {
	return tx.Delete(a).Error
}

func FindAutomationSource(tx *gorm.DB, organizationID, id uuid.UUID) (*AutomationSource, error) {
	var source AutomationSource
	err := tx.Where("organization_id = ? AND id = ?", organizationID, id).Take(&source).Error
	return &source, err
}

func FindAutomationSourceByArtifact(tx *gorm.DB, organizationID uuid.UUID, artifactID string) (*AutomationSource, error) {
	var source AutomationSource
	err := tx.Where("organization_id = ? AND artifact_id = ?", organizationID, artifactID).Take(&source).Error
	return &source, err
}

func ListAutomationSources(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]AutomationSource, error) {
	var sources []AutomationSource
	err := tx.Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).Find(&sources).Error
	return sources, err
}

// RefreshAutomationArtifactRevision advances the source only if it tracked the
// artifact before the factory refresh. Independent canvas edits stay protected.
func RefreshAutomationArtifactRevision(tx *gorm.DB, organizationID uuid.UUID, artifactID, previous, revision string) error {
	return tx.Model(&AutomationSource{}).
		Where("organization_id = ? AND artifact_id = ? AND artifact_revision = ?", organizationID, artifactID, previous).
		Update("artifact_revision", revision).Error
}

// LockAutomationFactory serializes YAML and legacy writes, including adoption
// when an automation does not have a source record yet.
func LockAutomationFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Factory, error) {
	return FindFactory(tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}), organizationID, factoryID)
}

func IsAutomationManagedArtifact(tx *gorm.DB, organizationID uuid.UUID, artifactID string) (bool, error) {
	_, err := FindAutomationSourceByArtifact(tx, organizationID, artifactID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}
