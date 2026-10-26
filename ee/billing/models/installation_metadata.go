package models

import (
	"fmt"
	"time"

	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// MaxPastDueGraceDays is the longest grace period an installation can set.
// Polar cancels a past-due subscription after its last retry on day 21.
const MaxPastDueGraceDays = 21

type installationBillingMetadata struct {
	ID               int `gorm:"primary_key"`
	PastDueGraceDays int
}

func (installationBillingMetadata) TableName() string { return "installation_metadata" }

func GetPastDueGraceDays(tx *gorm.DB) (int, error) {
	// Core creates the shared singleton with its general installation defaults.
	if _, err := coremodels.GetInstallationMetadata(tx); err != nil {
		return 0, err
	}
	var settings installationBillingMetadata
	if err := tx.Where("id = ?", 1).First(&settings).Error; err != nil {
		return 0, err
	}
	return settings.PastDueGraceDays, nil
}

func ValidatePastDueGraceDays(days int) error {
	if days < 0 || days > MaxPastDueGraceDays {
		return fmt.Errorf("past due grace must be between 0 and %d days", MaxPastDueGraceDays)
	}
	return nil
}

func UpdatePastDueGraceDays(tx *gorm.DB, days int) error {
	if err := ValidatePastDueGraceDays(days); err != nil {
		return err
	}
	if _, err := coremodels.GetInstallationMetadata(tx); err != nil {
		return err
	}
	return tx.Model(&installationBillingMetadata{}).Where("id = ?", 1).
		Updates(map[string]any{"past_due_grace_days": days, "updated_at": time.Now()}).Error
}
