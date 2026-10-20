package entitlements

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"testing"
)

func TestResolveCatalogAccessAndMarkup(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	var original models.PlanFeatureDefinition
	require.NoError(t, db.First(&original, "key = ?", "byo_key").Error)
	t.Cleanup(func() { require.NoError(t, models.UpdatePlanFeatureDefinition(db, &original)) })
	require.NoError(t, models.UpdatePlanFeatureDefinition(db, &models.PlanFeatureDefinition{Key: "byo_key", Label: "Bring your own API key", Gated: true}))
	seats, parallel, markup := 3, 2, 250
	p := models.BillingPlan{Key: "test_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", SeatLimit: &seats, ParallelTaskLimit: &parallel, MarkupBPS: &markup, CreditFrequency: "once", CreditExpiration: "never", CreditAmountCents: 1000}
	require.NoError(t, models.SaveBillingPlan(db, &p))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, p.Key, nil)
	require.NoError(t, err)
	resolved, err := Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 3, *resolved.SeatLimit)
	require.Equal(t, 2, *resolved.ParallelTaskLimit)
	require.Equal(t, 250, resolved.MarkupBPS)
	require.True(t, resolved.HostedExecutionAllowed)
	require.True(t, resolved.Features["byo_subscription"])
	require.False(t, resolved.Features["byo_key"])
	require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: true}))
	enabled, err := HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	p.Features = []string{"byo_key"}
	require.NoError(t, models.SaveBillingPlan(db, &p))
	require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: r.Organization.ID, FeatureKey: "byo_key", Enabled: false}))
	enabled, err = HasFeature(db, r.Organization.ID, "byo_key")
	require.NoError(t, err)
	require.True(t, enabled)
	zero := 0
	require.NoError(t, models.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &zero))
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Zero(t, resolved.MarkupBPS)
	p.Features = []string{}
	p.MarkupBPS = nil
	require.NoError(t, models.SaveBillingPlan(db, &p))
	require.NoError(t, models.UpsertOrganizationLLMMarkup(db, r.Organization.ID, nil))
	installation, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, installation.MarkupBPS, resolved.MarkupBPS)
	t.Setenv(models.BillingEnabledEnv, "no")
	resolved, err = Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.Nil(t, resolved.SeatLimit)
	require.Nil(t, resolved.ParallelTaskLimit)
	definitions, err := models.ListPlanFeatureDefinitions(db)
	require.NoError(t, err)
	for _, feature := range definitions {
		require.True(t, resolved.Features[feature.Key])
	}
}
