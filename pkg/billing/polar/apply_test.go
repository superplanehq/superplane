package polar

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__ApplyOrderPaidGrantsFaceValueOnce(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(org.ID, orderID, 2500)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.LLMCreditGrantKindTopup, grant.Kind)
	assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)

	settings, err := models.FindOrganizationLLMSettings(db, org.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.NotNil(t, settings.PolarCustomerID)
	assert.Equal(t, "cust_polar_1", *settings.PolarCustomerID)

	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(2500), summary.GrantMicros)
}

func Test__ApplyOrderPaidIgnoresNonPacks(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(7350), grant.AmountMicros)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(2300), grant.AmountMicros)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(7350), grant.AmountMicros)
}

func Test__ApplyOrderPaidIgnoresSubscriptionCycle(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	event := paidPackEvent(org.ID, uuid.NewString(), 2500)
	event.Data.BillingReason = "subscription_cycle"
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(10000), grant.AmountMicros)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(10000), grant.AmountMicros)
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

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)
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

	_, err = models.FindLLMCreditGrantByPolarOrderID(db, orderID)
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
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	refund := refundedPackEvent(org.ID, orderID, "refunded", 2500)
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(2500), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedCustomFullAndPartial(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidCustomPackEvent(org.ID, orderID, 7350), nil))
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedCustomPackEvent(org.ID, orderID, "partially_refunded", 7350, 2000), nil))
	partial, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(2000), partial.GrantMicros)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedCustomPackEvent(org.ID, orderID, "refunded", 7350, 7350), nil))
	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(7350), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedPartialDoesNotGoBelowZero(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "partially_refunded", 1000), nil))
	summary, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(1500), summary.GrantMicros)

	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refundedPackEvent(org.ID, orderID, "partially_refunded", 99999), nil))
	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedIgnoresNonPacks(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderRefundedUsesExistingGrantWithoutPackMetadata(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)

	refund := refundedPackEvent(org.ID, orderID, "refunded", 2500)
	refund.Data.Product.Metadata = map[string]any{"superplane_credit_pack": false}
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(2500), after.GrantMicros)
}

func Test__ApplyOrderRefundedDoesNotOverDebitConcurrently(t *testing.T) {
	org := setupBillingOrganization(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(2500), after.GrantMicros)
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

	before, err := models.DescribeOrganizationLLMCredit(db, org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), before.GrantMicros)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(org.ID, orderID, 2500), nil))
	require.NoError(t, ApplyOrderRefunded(context.Background(), db, refund, nil))

	after, err := models.DescribeOrganizationLLMCredit(db, org.ID)
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
	org, err := models.CreateOrganization("billing-"+uuid.NewString(), "")
	require.NoError(t, err)
	account, err := models.CreateAccount("Billing test", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	require.NoError(t, models.GrantWelcomeCredit(database.Conn(), org.ID, account.ID))
	return org
}
