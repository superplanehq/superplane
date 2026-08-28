package polar

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var ErrPermanentApply = errors.New("polar webhook cannot be applied")

type CreditPackLookup interface {
	GetCreditPack(ctx context.Context, productID string) (*Product, error)
}

func IsPermanentApplyError(err error) bool {
	return err != nil && errors.Is(err, ErrPermanentApply)
}

func permanentApplyError(reason string) error {
	return fmt.Errorf("%w: %s", ErrPermanentApply, reason)
}

func ApplyOrderEvent(ctx context.Context, tx *gorm.DB, event *OrderWebhookEvent, lookup CreditPackLookup) error {
	if event == nil {
		return permanentApplyError("order event is required")
	}
	switch event.Type {
	case orderPaidType:
		return ApplyOrderPaid(ctx, tx, event, lookup)
	case orderRefundedType:
		return ApplyOrderRefunded(ctx, tx, event, lookup)
	default:
		return nil
	}
}

func ApplyOrderPaid(ctx context.Context, tx *gorm.DB, event *OrderWebhookEvent, lookup CreditPackLookup) error {
	if event == nil {
		return permanentApplyError("order event is required")
	}
	isPack, err := orderIsCreditPack(ctx, event.Data, lookup)
	if err != nil {
		return err
	}
	if !isPack {
		return nil
	}
	if !shouldGrantPurchase(event.Data) {
		return nil
	}

	orgID, err := parseOrganizationID(event.Data.Customer.ExternalID)
	if err != nil {
		return err
	}
	if _, err := models.FindOrganizationByIDInTransaction(tx, orgID.String()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return permanentApplyError("organization not found")
		}
		return err
	}

	amountCents := purchasedFaceValueCents(event.Data)
	if amountCents <= 0 {
		return permanentApplyError("credit pack face value is missing")
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		if _, err := models.AddPolarLLMCreditGrant(inner, orgID, models.CentsToMicros(amountCents), event.Data.ID); err != nil {
			return err
		}
		if customerID := event.Data.Customer.ID; customerID != "" {
			if err := models.SetOrganizationPolarCustomerID(inner, orgID, customerID); err != nil {
				return err
			}
		}
		return nil
	})
}

func ApplyOrderRefunded(ctx context.Context, tx *gorm.DB, event *OrderWebhookEvent, lookup CreditPackLookup) error {
	if event == nil {
		return permanentApplyError("order event is required")
	}

	orgID, err := parseOrganizationID(event.Data.Customer.ExternalID)
	if err != nil {
		return err
	}

	grant, err := models.FindLLMCreditGrantByPolarOrderID(tx, event.Data.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return refundBeforeGrantError(ctx, event.Data, lookup)
		}
		return err
	}

	reverseMicros := refundReverseMicros(grant.AmountMicros, event.Data)
	refundID := polarRefundID(event.Data, reverseMicros)
	return models.ReversePolarOrderCredit(tx, orgID, event.Data.ID, reverseMicros, refundID)
}

func parseOrganizationID(externalID string) (uuid.UUID, error) {
	orgID, err := uuid.Parse(strings.TrimSpace(externalID))
	if err != nil {
		return uuid.Nil, permanentApplyError("order customer external id is not an organization id")
	}
	return orgID, nil
}

func orderIsCreditPack(ctx context.Context, data OrderData, lookup CreditPackLookup) (bool, error) {
	if data.Product.IsCreditPack() {
		return true, nil
	}
	if lookup == nil || strings.TrimSpace(data.Product.ID) == "" {
		return false, nil
	}
	pack, err := lookup.GetCreditPack(ctx, data.Product.ID)
	if err != nil {
		if errors.Is(err, ErrNotCreditPack) || IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return pack != nil, nil
}

// refundBeforeGrantError keeps Polar retrying a pack refund until order.paid
// records the grant. A 202 here would drop the refund. Non-pack refunds stay
// ignored so Polar does not disable the endpoint.
func refundBeforeGrantError(ctx context.Context, data OrderData, lookup CreditPackLookup) error {
	isPack, err := orderIsCreditPack(ctx, data, lookup)
	if err != nil {
		return err
	}
	if !isPack {
		return nil
	}
	return fmt.Errorf("polar credit grant for order %s is not recorded yet", data.ID)
}

func shouldGrantPurchase(data OrderData) bool {
	reason := strings.TrimSpace(data.BillingReason)
	if reason == billingReasonPurchase {
		return true
	}
	if reason == "" {
		return !data.Product.IsRecurring
	}
	return false
}

// purchasedFaceValueCents uses the price Polar charged, not the first catalog price.
func purchasedFaceValueCents(data OrderData) int64 {
	if amount := fixedPriceAmount(data.ProductPrice); amount > 0 {
		return amount
	}
	for _, item := range data.Items {
		if amount := fixedPriceAmount(item.ProductPrice); amount > 0 {
			return amount
		}
		if item.Amount > 0 {
			return item.Amount
		}
	}
	if amount := singleFixedCatalogPrice(data.Product.Prices); amount > 0 {
		return amount
	}
	return data.Product.FaceValueCents()
}

func fixedPriceAmount(price priceJSON) int64 {
	if price.AmountType != "" && price.AmountType != "fixed" {
		return 0
	}
	if price.PriceAmount > 0 {
		return price.PriceAmount
	}
	return 0
}

func singleFixedCatalogPrice(prices []priceJSON) int64 {
	amount := int64(0)
	found := 0
	for _, price := range prices {
		fixed := fixedPriceAmount(price)
		if fixed <= 0 {
			continue
		}
		found++
		amount = fixed
	}
	if found != 1 {
		return 0
	}
	return amount
}

// refundReverseMicros maps Polar refunded net (excluding tax) onto the SuperPlane
// pack face-value grant. A full Polar refund reverses the whole grant. A partial
// refund reverses min(grant, refunded_net in micros) and never more than the grant.
func refundReverseMicros(grantMicros int64, data OrderData) int64 {
	if grantMicros <= 0 {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(data.Status), "refunded") {
		return grantMicros
	}
	refundedNet := data.RefundedAmount
	if refundedNet <= 0 {
		return 0
	}
	refundedMicros := models.CentsToMicros(refundedNet)
	if refundedMicros > grantMicros {
		return grantMicros
	}
	return refundedMicros
}

func polarRefundID(data OrderData, reverseMicros int64) string {
	status := strings.ToLower(strings.TrimSpace(data.Status))
	if status == "refunded" {
		return data.ID + ":full"
	}
	return fmt.Sprintf("%s:%d", data.ID, reverseMicros)
}

func LogPermanentApply(event *OrderWebhookEvent, err error) {
	fields := log.Fields{"error": err}
	if event != nil {
		fields["polar_order_id"] = event.Data.ID
		fields["polar_event_type"] = event.Type
	}
	log.WithFields(fields).Error("ignored polar webhook that cannot be applied")
}
