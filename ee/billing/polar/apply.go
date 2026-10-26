package polar

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
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
	if productID := event.Data.productID(); productID != "" {
		_, err := billingmodels.FindBillingPlanByProduct(tx, productID)
		if err == nil {
			return resyncSubscriptionAfterCycle(ctx, tx, event.Data)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
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

	orgID, err := parseOrganizationID(event.Data)
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
	quote, err := creditPurchaseFromOrder(event.Data)
	if err != nil {
		return err
	}
	if quote != nil {
		amountCents = quote.CreditAmountCents
	}
	if amountCents <= 0 {
		return permanentApplyError("credit pack face value is missing")
	}

	var created bool
	err = tx.Transaction(func(inner *gorm.DB) error {
		var err error
		_, created, err = billingmodels.EnsureTopupLLMCreditGrant(inner, orgID, billingmodels.CentsToMicros(amountCents), event.Data.ID)
		if err != nil {
			return err
		}
		if customerID := event.Data.Customer.ID; customerID != "" {
			if err := billingmodels.SetOrganizationPolarCustomerID(inner, orgID, customerID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if created {
		logDiscountedCreditGrant(orgID, event.Data, amountCents)
	}
	return nil
}

// logDiscountedCreditGrant records discounted grants. The grant keeps the
// quoted credits, so a discount reduces revenue and needs an audit trail.
func logDiscountedCreditGrant(orgID uuid.UUID, order OrderData, grantedCents int64) {
	if order.DiscountAmount <= 0 {
		return
	}
	log.WithFields(log.Fields{
		"organization_id":      orgID.String(),
		"polar_order_id":       order.ID,
		"polar_discount_id":    order.DiscountID,
		"discount_amount":      order.DiscountAmount,
		"net_amount":           order.NetAmount,
		"granted_credit_cents": grantedCents,
	}).Info("granted credits for discounted polar order")
}

// resyncSubscriptionAfterCycle refreshes the subscription from Polar when a
// plan renewal order is paid. It recovers the period and the Included grant
// when a subscription event was skipped as stale or could not be parsed.
func resyncSubscriptionAfterCycle(ctx context.Context, tx *gorm.DB, data OrderData) error {
	if strings.TrimSpace(data.BillingReason) != billingReasonSubscriptionCycle {
		return nil
	}
	orgID, err := parseOrganizationID(data)
	if err != nil {
		return err
	}
	return SyncOrganizationSubscription(ctx, tx, orgID)
}

func ApplyOrderRefunded(ctx context.Context, tx *gorm.DB, event *OrderWebhookEvent, lookup CreditPackLookup) error {
	if event == nil {
		return permanentApplyError("order event is required")
	}

	orgID, err := parseOrganizationID(event.Data)
	if err != nil {
		return err
	}

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(tx, event.Data.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return refundBeforeGrantError(ctx, event.Data, lookup)
		}
		return err
	}

	reverseMicros, err := creditPurchaseRefundMicros(grant.AmountMicros, event.Data)
	if err != nil {
		return err
	}
	refundID := polarRefundID(event.Data, reverseMicros)
	return billingmodels.ReversePolarOrderCredit(tx, orgID, event.Data.ID, reverseMicros, refundID)
}

func creditPurchaseRefundMicros(grantMicros int64, order OrderData) (int64, error) {
	quote, err := creditPurchaseFromOrder(order)
	if err != nil {
		return 0, err
	}
	if quote == nil {
		return refundReverseMicros(grantMicros, order), nil
	}
	if billingmodels.CentsToMicros(quote.CreditAmountCents) != grantMicros {
		return 0, permanentApplyError("credit purchase metadata does not match the recorded grant")
	}
	if strings.EqualFold(strings.TrimSpace(order.Status), "refunded") {
		return grantMicros, nil
	}
	// Prorate against the amount paid. Settlement checked that net_amount is
	// the quoted subtotal less any discount.
	paid := order.NetAmount
	if order.RefundedAmount <= 0 || paid <= 0 {
		return 0, nil
	}
	refunded := min(order.RefundedAmount, paid)
	// Split division before multiplication to avoid overflowing large purchases.
	return grantMicros/paid*refunded + grantMicros%paid*refunded/paid, nil
}

func parseOrganizationID(data OrderData) (uuid.UUID, error) {
	orgID, err := uuid.Parse(data.organizationExternalID())
	if err != nil {
		return uuid.Nil, permanentApplyError("order customer external id is not an organization id")
	}
	return orgID, nil
}

func orderIsCreditPack(ctx context.Context, data OrderData, lookup CreditPackLookup) (bool, error) {
	// Checkout metadata is a snapshot. It remains valid after the admin edits
	// or deletes the pack, and does not require metadata on the Polar product.
	quote, err := creditPurchaseFromOrder(data)
	if err != nil {
		return false, err
	}
	if quote != nil {
		return true, nil
	}
	if data.Product.IsCreditPack() {
		return true, nil
	}
	for _, item := range data.Items {
		if item.Product.IsCreditPack() {
			return true, nil
		}
	}
	productID := data.productID()
	if lookup == nil || productID == "" {
		return false, nil
	}
	pack, err := lookup.GetCreditPack(ctx, productID)
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
// Custom (pay-what-you-want) packs grant the paid net amount, excluding tax.
func purchasedFaceValueCents(data OrderData) int64 {
	if orderHasCustomPrice(data) {
		return customPaidAmountCents(data)
	}
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

func orderHasCustomPrice(data OrderData) bool {
	if data.ProductPrice.isCustom() {
		return true
	}
	for _, item := range data.Items {
		if item.ProductPrice.isCustom() {
			return true
		}
	}
	return false
}

func customPaidAmountCents(data OrderData) int64 {
	if data.NetAmount > 0 {
		return data.NetAmount
	}
	for _, item := range data.Items {
		if item.Amount > 0 {
			return item.Amount
		}
	}
	if data.ProductPrice.PriceAmount > 0 {
		return data.ProductPrice.PriceAmount
	}
	return 0
}

func fixedPriceAmount(price priceJSON) int64 {
	if !price.isFixed() {
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
	refundedMicros := billingmodels.CentsToMicros(refundedNet)
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
