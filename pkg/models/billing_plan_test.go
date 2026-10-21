package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCatalogPlanAssignmentAndProtection(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	zero := 0
	plan := models.BillingPlan{Key: "custom_" + uuid.NewString()[:8], Name: "Custom", Category: "paid", Visibility: "private", MarkupBPS: &zero, CreditAmountCents: 1234, CreditFrequency: "once", CreditExpiration: "after_days"}
	require.Error(t, models.SaveBillingPlan(db, &plan))
	days := 30
	plan.CreditValidityDays = &days
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	assigned, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	require.True(t, assigned.IsActiveBusiness())
	require.Equal(t, plan.ID, assigned.PlanID)
	markup, err := models.ResolveOrganizationMarkupBPS(db, r.Organization.ID)
	require.NoError(t, err)
	require.Zero(t, markup)
	require.Error(t, models.DeleteBillingPlan(db, plan.ID))
	grants, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	for _, grant := range grants {
		require.NotEqual(t, models.LLMCreditGrantKindIncluded, grant.Kind)
	}
	original := plan.Key
	plan.Key = "changed"
	require.Error(t, models.SaveBillingPlan(db, &plan))
	plan.Key = original
	product := "product_" + uuid.NewString()
	plan.PolarProductID = &product
	plan.PriceCents = 3000
	plan.PerSeat = true
	require.NoError(t, models.SaveBillingPlan(db, &plan))
	stored, err := models.FindBillingPlan(db, plan.Key)
	require.NoError(t, err)
	require.Equal(t, int64(3000), stored.PriceCents)
	require.True(t, stored.PerSeat)
	plan.PerSeat = false
	require.ErrorIs(t, models.SaveBillingPlan(db, &plan), models.ErrCatalogPlan)
	plan.PerSeat = true
	plan.PriceCents = 2000
	require.ErrorIs(t, models.SaveBillingPlan(db, &plan), models.ErrCatalogPlan)
	manual, err := models.FindBillingPlan(db, "business_manual")
	require.NoError(t, err)
	require.ErrorIs(t, models.DeleteBillingPlan(db, manual.ID), models.ErrCatalogPlan)
	manual.Category = "trial"
	require.ErrorIs(t, models.SaveBillingPlan(db, manual), models.ErrCatalogPlan)
	manual.Category = "paid"
	manual.PolarProductID = &product
	require.ErrorIs(t, models.SaveBillingPlan(db, manual), models.ErrCatalogPlan)
}
