package polar

import (
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
	event := &OrderPaidEvent{
		Type: orderPaidType,
		Data: OrderPaidData{
			ID: orderID,
			Customer: OrderCustomer{
				ID:         "cust_polar_1",
				ExternalID: r.Organization.ID.String(),
			},
			Product: OrderProduct{
				ID: "prod_25",
				Metadata: map[string]any{
					"superplane_credit_pack": true,
				},
				Prices: []priceJSON{{AmountType: "fixed", PriceAmount: 2500}},
			},
		},
	}

	require.NoError(t, ApplyOrderPaid(db, event))
	require.NoError(t, ApplyOrderPaid(db, event))

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

	err = ApplyOrderPaid(db, &OrderPaidEvent{
		Data: OrderPaidData{
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
	})
	require.NoError(t, err)

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros, after.GrantMicros)
}
