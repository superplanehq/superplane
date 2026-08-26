package polar

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func ApplyOrderPaid(tx *gorm.DB, event *OrderPaidEvent) error {
	if event == nil {
		return fmt.Errorf("order event is required")
	}
	if !event.Data.Product.IsCreditPack() {
		return nil
	}

	orgID, err := uuid.Parse(event.Data.Customer.ExternalID)
	if err != nil {
		return fmt.Errorf("order customer external id is not an organization id")
	}

	amountCents := event.Data.faceValueCents()
	if amountCents <= 0 {
		return fmt.Errorf("credit pack face value is missing")
	}

	if _, err := models.FindOrganizationByIDInTransaction(tx, orgID.String()); err != nil {
		return err
	}

	if _, err := models.AddPolarLLMCreditGrant(tx, orgID, models.CentsToMicros(amountCents), event.Data.ID); err != nil {
		return err
	}
	if customerID := event.Data.Customer.ID; customerID != "" {
		if err := models.SetOrganizationPolarCustomerID(tx, orgID, customerID); err != nil {
			return err
		}
	}
	return nil
}

func (d OrderPaidData) faceValueCents() int64 {
	if amount := d.Product.FaceValueCents(); amount > 0 {
		return amount
	}
	return d.ProductPrice.PriceAmount
}
