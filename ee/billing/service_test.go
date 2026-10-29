package billing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	eebilling "github.com/superplanehq/superplane/ee/billing"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOrganizationPlanSummariesBatchReadsAndGraceExpiration(t *testing.T) {
	t.Setenv(billing.BillingEnabledEnv, "no")
	t.Cleanup(billing.Bind(eebilling.New(nil)))
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(tx, 7))
	custom := billingmodels.BillingPlan{Key: "batch_" + uuid.NewString()[:8], Name: "Private Team", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(tx, &custom))
	trial, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanTrial)
	require.NoError(t, err)
	now := time.Now()
	future, recent := now.Add(time.Hour), now.Add(-time.Hour)
	ids := make([]uuid.UUID, 60)
	expected := make(map[uuid.UUID]billing.OrganizationPlanSummary, len(ids))
	for i := range ids {
		org, err := models.CreateOrganizationInTransaction(tx, support.RandomName("batch-plan"), "")
		require.NoError(t, err)
		ids[i] = org.ID
		plan := billingmodels.OrganizationBillingPlan{OrganizationID: org.ID, PlanID: custom.ID, Plan: custom.Key, PlanSource: billingmodels.BillingPlanSourceAdmin}
		catalog := custom
		switch i % 3 {
		case 1:
			catalog = *trial
			plan.PlanID, plan.Plan, plan.TrialEndsAt = trial.ID, trial.Key, &future
		case 2:
			plan.PlanSource = billingmodels.BillingPlanSourcePolar
			plan.PolarSubscriptionStatus = billingmodels.PolarSubscriptionStatusPastDue
			plan.PastDueSince = &recent
		}
		require.NoError(t, tx.Omit("Catalog").Create(&plan).Error)
		expected[org.ID] = billing.OrganizationPlanSummary{Key: catalog.Key, Name: catalog.Name}
	}
	for _, count := range []int{3, len(ids)} {
		t.Run(fmt.Sprintf("%d organizations", count), func(t *testing.T) {
			queries := &planSummaryQueryCounter{Interface: tx.Logger}
			summaries, err := billing.ResolveOrganizationPlanSummaries(tx.Session(&gorm.Session{Logger: queries}), append(ids[:count:count], ids[0]))
			require.NoError(t, err)
			require.Len(t, summaries, count)
			for _, id := range ids[:count] {
				assert.Equal(t, expected[id], summaries[id])
			}
			assert.LessOrEqual(t, queries.count, 4, "billing rows, catalogs, and grace settings must not be read separately for each organization")
		})
	}
	// A later batch must use the current setting, and each expired row must
	// independently fall back to its remaining trial or None.
	require.NoError(t, tx.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", ids[5]).Update("trial_ends_at", future).Error)
	require.NoError(t, billingmodels.UpdatePastDueGraceDays(tx, 0))
	summaries, err := billing.ResolveOrganizationPlanSummaries(tx, ids[:6])
	require.NoError(t, err)
	none, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanNone)
	require.NoError(t, err)
	for id, catalog := range map[uuid.UUID]billingmodels.BillingPlan{ids[2]: *none, ids[5]: *trial} {
		assert.Equal(t, billing.OrganizationPlanSummary{Key: catalog.Key, Name: catalog.Name}, summaries[id])
		stored, err := billingmodels.FindOrganizationBillingPlan(tx, id)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, catalog.ID, stored.PlanID)
		assert.Equal(t, billingmodels.PolarSubscriptionStatusPastDue, stored.PolarSubscriptionStatus)
	}
}

type planSummaryQueryCounter struct {
	logger.Interface
	count int
}

func (l *planSummaryQueryCounter) Trace(ctx context.Context, begin time.Time, query func() (string, int64), err error) {
	l.count++
	l.Interface.Trace(ctx, begin, query, err)
}

func TestRuntimeInitializesOrganizationBilling(t *testing.T) {
	t.Setenv(billing.BillingEnabledEnv, "yes")
	t.Cleanup(billing.Bind(eebilling.New(nil)))
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	require.NoError(t, billing.Initialize(db))

	organization, err := models.CreateOrganization(support.RandomName("bound-billing"), "")
	require.NoError(t, err)
	plan, err := billingmodels.FindOrganizationBillingPlan(db, organization.ID)
	require.NoError(t, err)
	require.NotNil(t, plan, "core organization creation must initialize the bound billing implementation")
	assert.Equal(t, billingmodels.BillingPlanNone, plan.Plan)
	assert.Equal(t, billingmodels.BillingPlanSourceSystem, plan.PlanSource)
	assert.Nil(t, plan.TrialStartedAt)
	assert.Nil(t, plan.TrialEndsAt)
}

func TestInstallationSettingsUseCatalogTrialCredit(t *testing.T) {
	t.Cleanup(billing.Bind(eebilling.New(nil)))
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	legacy, err := billingmodels.GetInstallationLLMSettings(tx)
	require.NoError(t, err)
	legacy.WelcomeGrantCents = 1000
	legacy.MarkupBPS = 2500
	legacy.WarningThresholdBPS = 1500
	legacy.CreditPurchaseFeeBPS = 500
	legacy.CreditPurchaseFixedFeeCents = 50
	_, err = billingmodels.UpdateInstallationLLMSettings(tx, *legacy)
	require.NoError(t, err)

	trial, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanTrial)
	require.NoError(t, err)
	trial.TrialCreditCents = 7500
	require.NoError(t, billingmodels.SaveBillingPlan(tx, trial))

	settings, err := billing.GetInstallationSettings(tx)
	require.NoError(t, err)
	assert.Equal(t, billing.InstallationSettings{
		TrialCreditCents:            7500,
		MarkupBPS:                   2500,
		WarningThresholdBPS:         1500,
		CreditPurchaseFeeBPS:        500,
		CreditPurchaseFixedFeeCents: 50,
	}, settings)
}

func TestBoundRuntimeRecordsUsageAcrossBillingModes(t *testing.T) {
	for _, enabled := range []string{"yes", "no"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv(billing.BillingEnabledEnv, enabled)
			t.Cleanup(billing.Bind(eebilling.New(nil)))
			r := support.Setup(t)
			t.Cleanup(r.Close)
			db := database.DB(t.Context())
			usage := factoryUsage(t, r)
			usage.Model = "runtime-priced-model"
			usage.FundingSource = models.UsageFundingSourceHosted
			require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, r.Organization.ID, new(2500)))
			current, err := billingmodels.FindCurrentUsagePriceBook(db)
			require.NoError(t, err)
			var published *billingmodels.UsagePriceBook
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				var err error
				published, err = billingmodels.PublishUsagePriceBook(tx, []billingmodels.UsagePriceBookRate{{
					UsageKind:            models.UsageKindModel,
					Provider:             usage.Provider,
					MatchKey:             usage.Model,
					MatchMode:            billingmodels.UsagePriceBookMatchExact,
					InputCentsPerMillion: 321,
				}}, current.Version)
				return err
			}))
			t.Cleanup(func() {
				cleanupDB := database.DB(context.WithoutCancel(t.Context()))
				require.NoError(t, billingmodels.ActivateUsagePriceBook(cleanupDB, current.Version))
				require.NoError(t, billingmodels.DeleteUsagePriceBook(cleanupDB, published.Version))
				require.NoError(t, billingmodels.LoadCurrentPriceBook(cleanupDB))
			})
			pricebook.Reset()
			require.False(t, pricebook.IsPriced(usage.Provider, usage.Model))

			require.NoError(t, billing.Initialize(db))
			require.NoError(t, models.RecordUsage(db, usage))

			var recorded models.WorkspaceUsageEvent
			require.NoError(t, db.Where("node_execution_id = ?", usage.NodeExecutionID).First(&recorded).Error)
			assert.Equal(t, int64(3_210_000), recorded.ProviderCostMicros)
			assert.Equal(t, int64(4_012_500), recorded.CostMicros)
			assert.Equal(t, published.Version, recorded.PriceBookVersion)
			require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).
				Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
			if enabled == "yes" {
				support.AssignOpenSystemTrial(t, r.Organization.ID)
			}
			err = billing.AssertHostedRunAllowed(db, r.Organization.ID, nil)
			if enabled == "no" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, billing.ErrHostedCreditEmpty)
		})
	}
}

func TestMissingRuntimeRejectsCoreWorkflows(t *testing.T) {
	t.Setenv(billing.BillingEnabledEnv, "yes")
	t.Cleanup(billing.Bind(eebilling.New(nil)))
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())
	usage := factoryUsage(t, r)
	t.Cleanup(billing.Bind(nil))

	require.ErrorIs(t, billing.Initialize(db), billing.ErrImplementationMissing)
	name := support.RandomName("missing-billing")
	_, err := models.CreateOrganization(name, "")
	require.ErrorIs(t, err, billing.ErrImplementationMissing)
	var organizations int64
	require.NoError(t, db.Model(&models.Organization{}).Where("name = ?", name).Count(&organizations).Error)
	assert.Zero(t, organizations)
	require.ErrorIs(t, models.RecordUsage(db, usage), billing.ErrImplementationMissing)
	var events int64
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("node_execution_id = ?", usage.NodeExecutionID).Count(&events).Error)
	assert.Zero(t, events)
}

func factoryUsage(t *testing.T, resources *support.ResourceRegistry) models.WorkspaceUsageEventInput {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, resources.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, resources, factory.ID, "billing", "start")
	event := support.EmitCanvasEventForNode(t, canvas.ID, entry, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
	require.NoError(t, err)
	return models.WorkspaceUsageEventInput{
		OrganizationID:  resources.Organization.ID,
		CanvasRunID:     run.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}
}
