package testsupport

import (
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"gorm.io/gorm"
)

var seedUsagePriceBookVersions = []string{"2026-09-23.1"}

// RestoreSeedUsagePriceBooks restores the catalog used by backend fixtures.
func RestoreSeedUsagePriceBooks(tx *gorm.DB) error {
	if err := tx.Where("version NOT IN ?", seedUsagePriceBookVersions).Delete(&billingmodels.UsagePriceBookRate{}).Error; err != nil {
		return err
	}
	if err := tx.Where("version NOT IN ?", seedUsagePriceBookVersions).Delete(&billingmodels.UsagePriceBook{}).Error; err != nil {
		return err
	}
	if err := billingmodels.ActivateUsagePriceBook(tx, "2026-09-23.1"); err != nil {
		return err
	}
	return billingmodels.LoadCurrentPriceBook(tx)
}
