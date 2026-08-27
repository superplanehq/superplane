package polar

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ApplyOrderPaidGrantsFaceValueOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(r.Organization.ID, orderID, 2500)

	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.LLMCreditGrantKindPolar, grant.Kind)
	assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)

	settings, err := models.FindOrganizationLLMSettings(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotNil(t, settings)
	require.NotNil(t, settings.PolarCustomerID)
	assert.Equal(t, "cust_polar_1", *settings.PolarCustomerID)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(2500), summary.GrantMicros)
}

func Test__ApplyOrderPaidIgnoresNonPacks(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)

	err = ApplyOrderPaid(context.Background(), db, &OrderWebhookEvent{
		Type: orderPaidType,
		Data: OrderData{
			ID: uuid.NewString(),
			Customer: OrderCustomer{
				ID:         "cust_other",
				ExternalID: r.Organization.ID.String(),
			},
			Product: OrderProduct{
				Metadata: map[string]any{"superplane_credit_pack": false},
				Prices:   []priceJSON{{AmountType: "fixed", PriceAmount: 9900}},
			},
		},
	}, nil)
	require.NoError(t, err)

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderPaidUsesPurchasedPrice(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(r.Organization.ID, orderID, 2500)
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

func Test__ApplyOrderPaidIgnoresSubscriptionCycle(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)

	event := paidPackEvent(r.Organization.ID, uuid.NewString(), 2500)
	event.Data.BillingReason = "subscription_cycle"
	require.NoError(t, ApplyOrderPaid(context.Background(), db, event, nil))

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}

func Test__ApplyOrderPaidLooksUpMissingPackMetadata(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	event := paidPackEvent(r.Organization.ID, orderID, 2500)
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

func Test__ApplyOrderPaidPermanentErrors(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	err := ApplyOrderPaid(context.Background(), db, paidPackEvent(uuid.New(), uuid.NewString(), 2500), nil)
	require.Error(t, err)
	assert.True(t, IsPermanentApplyError(err))

	event := paidPackEvent(r.Organization.ID, uuid.NewString(), 2500)
	event.Data.Customer.ExternalID = "not-a-uuid"
	err = ApplyOrderPaid(context.Background(), db, event, nil)
	require.Error(t, err)
	assert.True(t, IsPermanentApplyError(err))
}

func Test__ApplyOrderRefundedFullAndIdempotent(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(r.Organization.ID, orderID, 2500), nil))
	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)

	refund := refundedPackEvent(r.Organization.ID, orderID, "refunded", 2500)
	require.NoError(t, ApplyOrderRefunded(db, refund))
	require.NoError(t, ApplyOrderRefunded(db, refund))

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(2500), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedPartialDoesNotGoBelowZero(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(r.Organization.ID, orderID, 2500), nil))

	require.NoError(t, ApplyOrderRefunded(db, refundedPackEvent(r.Organization.ID, orderID, "partially_refunded", 1000)))
	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(1500), summary.GrantMicros)

	require.NoError(t, ApplyOrderRefunded(db, refundedPackEvent(r.Organization.ID, orderID, "partially_refunded", 99999)))
	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents), after.GrantMicros)
	assert.GreaterOrEqual(t, after.RemainingMicros, int64(0))
}

func Test__ApplyOrderRefundedIgnoresNonPacks(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)

	err = ApplyOrderRefunded(db, &OrderWebhookEvent{
		Type: orderRefundedType,
		Data: OrderData{
			ID:     uuid.NewString(),
			Status: "refunded",
			Customer: OrderCustomer{
				ExternalID: r.Organization.ID.String(),
			},
			Product: OrderProduct{
				ID:       "prod_other",
				Metadata: map[string]any{"superplane_credit_pack": false},
			},
			RefundedAmount: 2500,
		},
	})
	require.NoError(t, err)

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
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
