package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	eebilling "github.com/superplanehq/superplane/ee/billing"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestListOrganizationsByPlanBeyondPostgresParameterLimit(t *testing.T) {
	t.Cleanup(billing.Bind(eebilling.New(nil)))
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	plan, err := billingmodels.FindBillingPlan(tx, billingmodels.BillingPlanNone)
	require.NoError(t, err)
	const count = 65_536
	prefix := "large-plan-" + uuid.NewString() + "-"
	require.NoError(t, tx.Exec(`
		WITH inserted AS (
			INSERT INTO organizations (name, slug)
			SELECT ? || lpad(n::text, 5, '0'), ? || n::text
			FROM generate_series(1, ?) AS n
			RETURNING id
		)
		INSERT INTO organization_billing_plans (organization_id, plan, plan_id, plan_source)
		SELECT id, ?, ?, 'admin' FROM inserted
	`, prefix, prefix, count, plan.Key, plan.ID).Error)

	var pinned models.Organization
	require.NoError(t, tx.Where("name = ?", prefix+"00001").First(&pinned).Error)
	page, total, matchTotal, err := models.ListUnpinnedOrganizations(
		tx, models.OrganizationListFilter{Search: prefix, PlanKey: plan.Key},
		1, count-2, "name", "asc", []uuid.UUID{pinned.ID},
	)
	require.NoError(t, err)
	require.Equal(t, int64(count), matchTotal)
	require.Equal(t, int64(count-1), total)
	require.Len(t, page, 1)
	require.Equal(t, prefix+"65536", page[0].Name)
}
