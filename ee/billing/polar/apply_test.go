package polar

import (
	"context"
	"testing"

	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	eebilling "github.com/superplanehq/superplane/ee/billing"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func init() { billing.Bind(eebilling.New(NewRuntime())) }

func Test__ApplyOrderPaidGrantsFaceValueOnce(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.LLMCreditGrantKindTopup, grant.Kind)
	assert.Equal(t, billingmodels.CentsToMicros(2500), grant.AmountMicros)

	settings, err := billingmodels.FindOrganizationLLMSettings(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.NotNil(t, settings.PolarCustomerID)
	assert.Equal(t, "cust_polar_1", *settings.PolarCustomerID)

	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(2500), summary.GrantMicros)
}

func Test__ApplyOrderPaidIgnoresNonPacks(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	err = ApplyOrderPaid(context.Background(), db, &OrderWebhookEvent{
		Type: orderPaidType,
		Data: OrderData{
			ID: uuid.NewString(),
			Customer: OrderCustomer{
				ID:         "cust_other",
				ExternalID: org.ID.String(),
			},
			Product: OrderProduct{
				Metadata: map[string]any{"superplane_credit_pack": false},
				Prices:   []priceJSON{{AmountType: "fixed", PriceAmount: 9900}},
			},
		},
	}, nil)
	require.NoError(t, err)

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderPaidGrantsCustomPaidAmount(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidCustomPackEvent(org.ID, orderID, 7350)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(7350), grant.AmountMicros)
}

func TestApplyOrderPaidGrantsCreditsWithoutProcessingFeeOnce(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidFeePackEvent(org.ID, orderID)

	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), grant.AmountMicros)
}

func TestApplyOrderPaidRejectsInvalidCreditPurchaseMetadata(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	for _, test := range []struct {
		name   string
		change func(*OrderData)
	}{
		{name: "missing version", change: func(order *OrderData) { delete(order.Metadata, "superplane_credit_purchase_version") }},
		{name: "unsupported version", change: func(order *OrderData) { order.Metadata["superplane_credit_purchase_version"] = "2" }},
		{name: "missing credit amount", change: func(order *OrderData) { delete(order.Metadata, "superplane_credit_amount_cents") }},
		{name: "fractional credit amount", change: func(order *OrderData) { order.Metadata["superplane_credit_amount_cents"] = "10000.5" }},
		{name: "numeric credit amount", change: func(order *OrderData) { order.Metadata["superplane_credit_amount_cents"] = 10000 }},
		{name: "zero credits", change: func(order *OrderData) { order.Metadata["superplane_credit_amount_cents"] = "0" }},
		{name: "credits exceed price", change: func(order *OrderData) { order.Metadata["superplane_credit_amount_cents"] = "10580" }},
		{name: "missing subtotal", change: func(order *OrderData) { delete(order.Metadata, "superplane_credit_subtotal_cents") }},
		{name: "amount overflow", change: func(order *OrderData) { order.Metadata["superplane_credit_subtotal_cents"] = "9223372036854775808" }},
		{name: "wrong paid amount", change: func(order *OrderData) { order.NetAmount = 10578 }},
		{name: "wrong currency", change: func(order *OrderData) { order.Currency = "eur" }},
		{name: "wrong subtotal", change: func(order *OrderData) { discountOrder(order, 10578, 0) }},
		{name: "net ignores discount", change: func(order *OrderData) {
			discountOrder(order, 10579, 1000)
			order.NetAmount = 10579
		}},
		{name: "negative discount", change: func(order *OrderData) { discountOrder(order, 10579, -100) }},
		{name: "discount exceeds subtotal", change: func(order *OrderData) { discountOrder(order, 10579, 10580) }},
		{name: "discount without subtotal", change: func(order *OrderData) {
			discountOrder(order, 10579, 1000)
			order.SubtotalAmount = nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			event := paidFeePackEvent(org.ID, uuid.NewString())
			test.change(&event.Data)
			err := ApplyOrderPaid(t.Context(), db, event, nil)
			require.Error(t, err)
			assert.True(t, IsPermanentApplyError(err))
			_, err = billingmodels.FindLLMCreditGrantByPolarOrderID(db, event.Data.ID)
			require.Error(t, err)
		})
	}
}

func TestApplyOrderPaidGrantsQuotedCreditsForDiscountedPurchaseOnce(t *testing.T) {
	for _, test := range []struct {
		name       string
		discount   int64
		concurrent bool
	}{
		{name: "partial discount", discount: 1000},
		{name: "full discount", discount: 10579},
		{name: "concurrent deliveries", discount: 1000, concurrent: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			db := database.Conn()
			logs := logtest.NewGlobal()
			t.Cleanup(logs.Reset)
			event := paidFeePackEvent(org.ID, uuid.NewString())
			discountOrder(&event.Data, 10579, test.discount)

			if test.concurrent {
				errs := make(chan error, 2)
				for range 2 {
					go func() {
						errs <- ApplyOrderPaid(t.Context(), db, event, nil)
					}()
				}
				require.NoError(t, <-errs)
				require.NoError(t, <-errs)
			}
			require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
			require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))

			grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, event.Data.ID)
			require.NoError(t, err)
			assert.Equal(t, billingmodels.CentsToMicros(10000), grant.AmountMicros)
			summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
			require.NoError(t, err)
			assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(10000), summary.GrantMicros)

			entries := logs.AllEntries()
			require.Len(t, entries, 1, "webhook replays must not log another discounted grant")
			entry := entries[0]
			assert.Equal(t, event.Data.ID, entry.Data["polar_order_id"])
			assert.Equal(t, "disc_1", entry.Data["polar_discount_id"])
			assert.Equal(t, test.discount, entry.Data["discount_amount"])
		})
	}
}

func TestApplyOrderPaidAcceptsUndiscountedOrderWithSubtotal(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	discountOrder(&event.Data, 10579, 0)

	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), grant.AmountMicros)
}

func TestApplyOrderRefundedProratesDiscountedPurchaseAgainstPaidAmount(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	discountOrder(&event.Data, 10579, 579)
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	event.Type = orderRefundedType

	event.Data.Status = "partially_refunded"
	event.Data.RefundedAmount = 5000
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(5000), reversed, "half of the paid amount reverses half of the grant")

	event.Data.Status = "refunded"
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err = billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), reversed)
}

func TestApplyOrderRefundedHandlesFullyDiscountedPurchase(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	discountOrder(&event.Data, 10579, 10579)
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	event.Type = orderRefundedType

	event.Data.Status = "partially_refunded"
	event.Data.RefundedAmount = 1
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Zero(t, reversed)

	event.Data.Status = "refunded"
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err = billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), reversed)
}

func TestApplyOrderRefundedProratesFeePurchaseCumulatively(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	event.Type = orderRefundedType
	event.Data.Status = "partially_refunded"

	for _, test := range []struct {
		refundedCents int64
		reverseMicros int64
	}{
		{refundedCents: 1, reverseMicros: 9452},
		{refundedCents: 5289, reverseMicros: 49995273},
		{refundedCents: 5289, reverseMicros: 49995273},
	} {
		event.Data.RefundedAmount = test.refundedCents
		require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
		reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
		require.NoError(t, err)
		assert.Equal(t, test.reverseMicros, reversed)
	}

	event.Data.Status = "refunded"
	event.Data.RefundedAmount = 0
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), reversed)
}

func TestApplyOrderRefundedProratesLargePurchaseWithoutOverflow(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	event.Data.Metadata["superplane_credit_amount_cents"] = "90000000"
	event.Data.Metadata["superplane_credit_subtotal_cents"] = "99999999"
	event.Data.NetAmount = 99999999
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	event.Type = orderRefundedType
	event.Data.Status = "partially_refunded"
	event.Data.RefundedAmount = 99999998
	require.NoError(t, ApplyOrderRefunded(t.Context(), db, event, nil))
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(899999990999), reversed)
}

func TestApplyOrderRefundedRejectsMetadataThatDiffersFromGrant(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	event := paidFeePackEvent(org.ID, uuid.NewString())
	require.NoError(t, ApplyOrderPaid(t.Context(), db, event, nil))
	event.Type = orderRefundedType
	event.Data.Status = "partially_refunded"
	event.Data.RefundedAmount = 1000
	event.Data.Metadata["superplane_credit_amount_cents"] = "9999"
	err := ApplyOrderRefunded(t.Context(), db, event, nil)
	require.Error(t, err)
	assert.True(t, IsPermanentApplyError(err))
	reversed, err := billingmodels.PolarRefundMicrosForOrder(db, event.Data.ID)
	require.NoError(t, err)
	assert.Zero(t, reversed)
}

func Test__ApplyOrderPaidUsesPurchasedPrice(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)
	event.Data.Product.Prices = []priceJSON{
		{AmountType: "fixed", PriceAmount: 2500},
		{AmountType: "fixed", PriceAmount: 2300},
	}
	event.Data.ProductPrice = priceJSON{AmountType: "fixed", PriceAmount: 2300}

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(2300), grant.AmountMicros)
}

func Test__ApplyOrderPaidGrantsFixedPriceWhenCatalogAlsoHasCustom(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)
	event.Data.NetAmount = 2000
	event.Data.Product.Prices = []priceJSON{
		{AmountType: "fixed", PriceAmount: 2500},
		{AmountType: "custom"},
	}
	event.Data.ProductPrice = priceJSON{AmountType: "fixed", PriceAmount: 2500}

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(2500), grant.AmountMicros)
}

func Test__ApplyOrderPaidGrantsCustomAmountWhenMixedCatalogSelectsCustom(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidCustomPackEvent(org.ID, orderID, 7350)
	event.Data.Product.Prices = []priceJSON{
		{AmountType: "fixed", PriceAmount: 2500},
		{AmountType: "custom"},
	}

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(7350), grant.AmountMicros)
}

func Test__ApplyOrderPaidIgnoresSubscriptionCycle(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	event := paidPackEvent(org.ID, uuid.NewString(), 2500)
	event.Data.BillingReason = "subscription_cycle"
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderPaidUsesExternalCustomerID(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 10000)
	event.Data.Customer.ExternalID = ""
	event.Data.ExternalCustomerID = org.ID.String()

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), grant.AmountMicros)
}

func Test__ApplyOrderPaidUsesItemPackMetadata(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 10000)
	event.Data.Product = OrderProduct{ID: "prod_100"}
	event.Data.ProductPrice = priceJSON{}
	event.Data.Items = []orderItemJSON{{
		Amount:       10000,
		ProductPrice: priceJSON{AmountType: "fixed", PriceAmount: 10000},
		Product: OrderProduct{
			ID: "prod_100",
			Metadata: map[string]any{
				"superplane_credit_pack": true,
			},
		},
	}}

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(10000), grant.AmountMicros)
}

func Test__ApplyOrderPaidLooksUpMissingPackMetadata(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)
	event.Data.Product.Metadata = nil
	lookup := creditPackLookupFunc(func(_ context.Context, productID string) (*Product, error) {
		assert.Equal(t, "prod_25", productID)
		return &Product{ID: productID, AmountCents: 2500}, nil
	})

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, lookup))

	grant, err := billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(2500), grant.AmountMicros)
}

func Test__ApplyOrderPaidRetriesWhenAPIVersionIsUnsupported(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)
	event.Data.Product.Metadata = nil
	lookup := creditPackLookupFunc(func(_ context.Context, _ string) (*Product, error) {
		return nil, ErrUnsupportedAPIVersion
	})

	err := ApplyOrderPaid(context.Background(), db, event, lookup)
	require.Error(t, err)
	assert.True(t, IsUnsupportedAPIVersion(err))
	assert.False(t, IsPermanentApplyError(err))

	_, err = billingmodels.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.Error(t, err)
}

func Test__ApplyOrderPaidPermanentErrors(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()

	err := ApplyOrderPaid(context.Background(), db, paidPackEvent(uuid.New(), uuid.NewString(), 2500), nil)
	require.Error(t, err)
	assert.True(t, IsPermanentApplyError(err))

	event := paidPackEvent(org.ID, uuid.NewString(), 2500)
	event.Data.Customer.ExternalID = "not-a-uuid"
	err = ApplyOrderPaid(context.Background(), db, event, nil)
	require.Error(t, err)
	assert.True(t, IsPermanentApplyError(err))
}

func Test__ApplyOrderRefundedFullAndIdempotent(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	refund := refundedPackEvent(org.ID, orderID, "refunded", 2500)
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-billingmodels.CentsToMicros(2500), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedCustomFullAndPartial(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidCustomPackEvent(org.ID, orderID, 7350), nil))
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedCustomPackEvent(org.ID, orderID, "partially_refunded", 7350, 2000), nil))
	partial, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-billingmodels.CentsToMicros(2000), partial.GrantMicros)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedCustomPackEvent(org.ID, orderID, "refunded", 7350, 7350), nil))
	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-billingmodels.CentsToMicros(7350), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedPartialDoesNotGoBelowZero(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "partially_refunded", 1000), nil))
	summary, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents)+billingmodels.CentsToMicros(1500), summary.GrantMicros)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "partially_refunded", 99999), nil))
	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedIgnoresNonPacks(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	err = ApplyOrderRefunded(context.Background(), db, &OrderWebhookEvent{
		Type: orderRefundedType,
		Data: OrderData{
			ID:     uuid.NewString(),
			Status: "refunded",
			Customer: OrderCustomer{
				ExternalID: org.ID.String(),
			},
			Product: OrderProduct{
				ID:       "prod_other",
				Metadata: map[string]any{"superplane_credit_pack": false},
			},
			RefundedAmount: 2500,
		},
	}, nil)
	require.NoError(t, err)

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderRefundedUsesExistingGrantWithoutPackMetadata(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	refund := refundedPackEvent(org.ID, orderID, "refunded", 2500)
	refund.Data.Product.Metadata = map[string]any{"superplane_credit_pack": false}
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-billingmodels.CentsToMicros(2500), after.GrantMicros)
}

func Test__ApplyOrderRefundedDoesNotOverDebitConcurrently(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	errs := make(chan error, 2)
	go func() {
		errs <- ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "refunded", 2500), nil)
	}()
	go func() {
		errs <- ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "partially_refunded", 1000), nil)
	}()
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-billingmodels.CentsToMicros(2500), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedRetriesUntilGrantExists(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	refund := refundedPackEvent(org.ID, orderID, "refunded", 2500)

	err := ApplyOrderRefunded(context.Background(), db, refund, nil)
	require.Error(t, err)
	assert.False(t, IsPermanentApplyError(err))

	before, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, billingmodels.CentsToMicros(billingmodels.SignupCreditCents), before.GrantMicros)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := billingmodels.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func paidPackEvent(orgID uuid.UUID, orderID string, amountCents int64) *OrderWebhookEvent {
	return &OrderWebhookEvent{
		Type: orderPaidType,
		Data: OrderData{
			ID:            orderID,
			BillingReason: billingReasonPurchase,
			Customer: OrderCustomer{
				ID:         "cust_polar_1",
				ExternalID: orgID.String(),
			},
			Product: OrderProduct{
				ID: "prod_25",
				Metadata: map[string]any{
					"superplane_credit_pack": true,
				},
				Prices: []priceJSON{{AmountType: "fixed", PriceAmount: amountCents}},
			},
			ProductPrice: priceJSON{AmountType: "fixed", PriceAmount: amountCents},
		},
	}
}

func paidFeePackEvent(orgID uuid.UUID, orderID string) *OrderWebhookEvent {
	event := paidPackEvent(orgID, orderID, 10579)
	event.Data.Currency = "usd"
	event.Data.NetAmount = 10579
	event.Data.Metadata = map[string]any{
		"superplane_credit_purchase_version": "1",
		"superplane_credit_amount_cents":     "10000",
		"superplane_credit_subtotal_cents":   "10579",
	}
	return event
}

func discountOrder(order *OrderData, subtotalCents, discountCents int64) {
	order.SubtotalAmount = &subtotalCents
	order.DiscountAmount = discountCents
	order.DiscountID = "disc_1"
	order.NetAmount = subtotalCents - discountCents
}

func paidCustomPackEvent(orgID uuid.UUID, orderID string, paidCents int64) *OrderWebhookEvent {
	event := paidPackEvent(orgID, orderID, paidCents)
	event.Data.NetAmount = paidCents
	event.Data.Product.ID = "prod_custom"
	event.Data.Product.Prices = []priceJSON{{AmountType: "custom", PriceAmount: 0}}
	event.Data.ProductPrice = priceJSON{AmountType: "custom"}
	event.Data.Items = []orderItemJSON{{
		Amount:       paidCents,
		ProductPrice: priceJSON{AmountType: "custom"},
		Product: OrderProduct{
			ID: "prod_custom",
			Metadata: map[string]any{
				"superplane_credit_pack": true,
			},
		},
	}}
	return event
}

func refundedCustomPackEvent(orgID uuid.UUID, orderID, status string, paidCents, refundedNet int64) *OrderWebhookEvent {
	event := paidCustomPackEvent(orgID, orderID, paidCents)
	event.Type = orderRefundedType
	event.Data.Status = status
	event.Data.RefundedAmount = refundedNet
	return event
}

func refundedPackEvent(orgID uuid.UUID, orderID, status string, refundedNet int64) *OrderWebhookEvent {
	event := paidPackEvent(orgID, orderID, 2500)
	event.Type = orderRefundedType
	event.Data.Status = status
	event.Data.RefundedAmount = refundedNet
	event.Data.NetAmount = 2500
	return event
}

type creditPackLookupFunc func(ctx context.Context, productID string) (*Product, error)

func (f creditPackLookupFunc) GetCreditPack(ctx context.Context, productID string) (*Product, error) {
	return f(ctx, productID)
}

func setupBillingOrganization(t *testing.T) *models.Organization {
	t.Helper()
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	catalog, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanBusiness)
	require.NoError(t, err)
	originalProduct := catalog.PolarProductID
	if originalProduct != nil {
		originalProduct = new(*originalProduct)
	}
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Model(catalog).Update("polar_product_id", originalProduct).Error)
	})
	require.NoError(t, db.Model(catalog).Update("polar_product_id", nil).Error)
	org, err := models.CreateOrganization("billing-"+uuid.NewString(), "")
	require.NoError(t, err)
	account, err := models.CreateAccount("Billing test", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	require.NoError(t, billingmodels.GrantWelcomeCredit(database.Conn(), org.ID, account.ID))
	support.AssignOpenSystemTrial(t, org.ID)
	return org
}
