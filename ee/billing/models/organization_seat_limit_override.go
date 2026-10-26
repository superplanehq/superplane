package models

import (
	"errors"

	"github.com/google/uuid"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrganizationSeatLimitOverride struct {
	OrganizationID uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	SeatLimit      *int      `json:"seat_limit"`
}

func FindOrganizationSeatLimitOverride(tx *gorm.DB, orgID uuid.UUID) (*OrganizationSeatLimitOverride, error) {
	var override OrganizationSeatLimitOverride
	err := tx.First(&override, "organization_id = ?", orgID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

func SetOrganizationSeatLimitOverride(tx *gorm.DB, orgID uuid.UUID, limit *int) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, orgID); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}}, DoUpdates: clause.AssignmentColumns([]string{"seat_limit"})}).Create(&OrganizationSeatLimitOverride{OrganizationID: orgID, SeatLimit: limit}).Error
	})
}

func DeleteOrganizationSeatLimitOverride(tx *gorm.DB, orgID uuid.UUID) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if _, err := coremodels.LockOrganization(tx, orgID); err != nil {
			return err
		}
		return tx.Delete(&OrganizationSeatLimitOverride{}, "organization_id = ?", orgID).Error
	})
}
