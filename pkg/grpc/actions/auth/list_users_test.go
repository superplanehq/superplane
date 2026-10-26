package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListUsers(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns role assignments", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), true, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)

		for _, user := range resp.Users {
			assert.NotEmpty(t, user.Metadata.Id)
			assert.NotEmpty(t, user.Status.Roles)
			assert.NotEmpty(t, user.Status.AccountProviders)
			for _, role := range user.Status.Roles {
				assert.NotEmpty(t, role)
				assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, role.DomainType)
				assert.Equal(t, r.Organization.ID.String(), role.DomainId)
			}
		}
	})

	t.Run("returns no role assignments", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), false, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Empty(t, resp.Users[0].Status.Roles)
	})
}

func TestListUsersSeatMetadata(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	limit := 3
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), r.Organization.ID, &limit))
	for _, roles := range []bool{false, true} {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), roles, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp.SeatLimit)
		require.EqualValues(t, 3, *resp.SeatLimit)
		require.Len(t, resp.Users, 1)
	}
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), r.Organization.ID, nil))
	resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), false, r.AuthService)
	require.NoError(t, err)
	require.Nil(t, resp.SeatLimit)

	maximum := 50
	plan := billingmodels.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: &maximum, PolarProductID: new("prod_" + uuid.NewString()), CreditFrequency: "none", CreditExpiration: "never"}
	db := database.Conn()
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err = billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ID: "sub_" + uuid.NewString(), ProductID: *plan.PolarProductID,
		Status: billingmodels.PolarSubscriptionStatusActive, PurchasedSeats: new(3),
	}, nil)
	require.NoError(t, err)
	for _, quantity := range []*int{&limit, nil} {
		require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", quantity).Error)
		for _, disabled := range []bool{false, true} {
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			if disabled {
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			}
			for _, roles := range []bool{false, true} {
				resp, err := ListUsers(t.Context(), models.DomainTypeOrganization, r.Organization.ID.String(), roles, r.AuthService)
				require.NoError(t, err)
				if disabled {
					require.Nil(t, resp.SeatLimit)
					continue
				}
				expected := int32(0)
				if quantity != nil {
					expected = int32(*quantity)
				}
				require.Equal(t, &expected, resp.SeatLimit)
			}
		}
	}
}
