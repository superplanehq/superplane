package models_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestResolveOrganizationParallelTaskLimitUsesEffectivePlanAndCallerTransaction(t *testing.T) {
	for _, state := range []string{"trial", "expired_trial", "expired_subscription", "unlimited", "disabled"} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			t.Setenv(models.BillingEnabledEnv, "yes")
			rollback := errors.New("rollback fixture")
			err := db.Transaction(func(tx *gorm.DB) error {
				// Uncommitted catalog edits must be visible to the resolver.
				require.NoError(t, tx.Model(&models.BillingPlan{}).Where("key = ?", "none").Update("parallel_task_limit", 2).Error)
				require.NoError(t, tx.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", 4).Error)
				want := 4
				switch state {
				case "expired_trial":
					expireOrganizationTrial(t, tx, r.Organization.ID)
					want = 2
				case "expired_subscription":
					require.NoError(t, tx.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
						"plan_id": tx.Model(&models.BillingPlan{}).Select("id").Where("key = ?", "business_legacy"),
						"plan":    models.BillingPlanBusiness, "plan_source": models.BillingPlanSourcePolar,
						"polar_subscription_status": models.PolarSubscriptionStatusActive,
						"cancel_at_period_end":      true, "current_period_end": time.Now().Add(-time.Hour), "trial_ends_at": time.Now().Add(-2 * time.Hour),
					}).Error)
					want = 2
				case "unlimited":
					require.NoError(t, tx.Model(&models.BillingPlan{}).Where("key = ?", "trial").Update("parallel_task_limit", nil).Error)
				case "disabled":
					t.Setenv(models.BillingEnabledEnv, "no")
				}
				limit, err := models.ResolveOrganizationParallelTaskLimit(tx, r.Organization.ID)
				require.NoError(t, err)
				if state == "unlimited" || state == "disabled" {
					require.Nil(t, limit)
				} else {
					require.Equal(t, &want, limit)
				}
				return rollback
			})
			require.ErrorIs(t, err, rollback)
		})
	}
}

func TestResolveOrganizationParallelTaskLimitPropagatesDatabaseErrors(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Rollback().Error)
	_, err := models.ResolveOrganizationParallelTaskLimit(tx, r.Organization.ID)
	require.Error(t, err)
}
