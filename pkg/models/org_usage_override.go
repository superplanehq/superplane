package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type OrgUsageOverride struct {
	ID                    uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	OrganizationID        uuid.UUID
	MaxOrgsPerAccount     *int
	MaxCanvasesPerOrg     *int
	MaxNodesPerCanvas     *int
	MaxUsersPerOrg        *int
	MaxIntegrationsPerOrg *int
	MaxEventsPerMonth     *int
	RetentionDays         *int
	IsUnlimited           bool
	CreatedAt             *time.Time
	UpdatedAt             *time.Time
}

func FindOrgUsageOverride(orgID uuid.UUID) (*OrgUsageOverride, error) {
	return FindOrgUsageOverrideInTransaction(database.Conn(), orgID)
}

func FindOrgUsageOverrideInTransaction(tx *gorm.DB, orgID uuid.UUID) (*OrgUsageOverride, error) {
	var override OrgUsageOverride
	err := tx.Where("organization_id = ?", orgID).First(&override).Error
	if err != nil {
		return nil, err
	}
	return &override, nil
}
