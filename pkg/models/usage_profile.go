package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type UsageProfile struct {
	ID                   uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name                 string
	MaxOrgsPerAccount    int
	MaxCanvasesPerOrg    int
	MaxNodesPerCanvas    int
	MaxUsersPerOrg       int
	MaxIntegrationsPerOrg int
	MaxEventsPerMonth    int
	RetentionDays        int
	CreatedAt            *time.Time
	UpdatedAt            *time.Time
}

func FindUsageProfileByName(name string) (*UsageProfile, error) {
	return FindUsageProfileByNameInTransaction(database.Conn(), name)
}

func FindUsageProfileByNameInTransaction(tx *gorm.DB, name string) (*UsageProfile, error) {
	var profile UsageProfile
	err := tx.Where("name = ?", name).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func FindUsageProfileByID(id uuid.UUID) (*UsageProfile, error) {
	return FindUsageProfileByIDInTransaction(database.Conn(), id)
}

func FindUsageProfileByIDInTransaction(tx *gorm.DB, id uuid.UUID) (*UsageProfile, error) {
	var profile UsageProfile
	err := tx.Where("id = ?", id).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
