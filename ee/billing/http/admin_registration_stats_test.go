package billinghttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestAdminRegistrationStats(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.DB(context.WithoutCancel(t.Context()))
	start := time.Now().UTC().AddDate(0, 0, -100).Truncate(time.Second)
	end := start.AddDate(0, 0, 1)
	ids := []uuid.UUID{}
	t.Cleanup(func() {
		require.NoError(t, db.Where("organization_id IN ?", ids).Delete(&models.WorkspaceUsageEvent{}).Error)
		require.NoError(t, db.Where("organization_id IN ?", ids).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
		require.NoError(t, db.Where("organization_id IN ?", ids).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, db.Unscoped().Where("organization_id IN ?", ids).Delete(&models.User{}).Error)
		require.NoError(t, db.Unscoped().Where("id IN ?", ids).Delete(&models.Organization{}).Error)
	})
	for i := 0; i < 9; i++ {
		created := start.Add(time.Duration(i) * time.Minute)
		if i == 6 {
			created = end
		}
		if i == 7 {
			created = start.Add(-time.Second)
		}
		org := models.Organization{ID: uuid.New(), Name: "Stats organization", Slug: uuid.NewString(), CreatedAt: &created}
		require.NoError(t, db.Create(&org).Error)
		ids = append(ids, org.ID)
		if i == 8 {
			require.NoError(t, db.Delete(&org).Error)
			continue
		}
		owner, err := models.CreateUserInTransaction(db, org.ID, r.Account.ID, r.Account.Email, r.Account.Name)
		require.NoError(t, err)
		require.NoError(t, models.SetUserIsOwner(db, owner.ID, true))
		expires := start.Add(time.Hour)
		if i == 5 {
			expires = time.Now().Add(time.Hour)
			plan, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanTrial)
			require.NoError(t, err)
			require.NoError(t, db.Create(&billingmodels.OrganizationBillingPlan{OrganizationID: org.ID, PlanID: plan.ID, Plan: plan.Key, PlanSource: "admin", TrialEndsAt: &expires}).Error)
			for _, kind := range []string{"topup", "admin"} {
				require.NoError(t, db.Create(&billingmodels.OrganizationLLMCreditGrant{ID: uuid.New(), OrganizationID: org.ID, Kind: kind, AmountMicros: 15000000, CreatedAt: start}).Error)
			}
		}
		require.NoError(t, db.Create(&billingmodels.OrganizationLLMCreditGrant{
			ID: uuid.New(), OrganizationID: org.ID, Kind: "welcome", AmountMicros: 10000000, CreatedAt: start, ExpiresAt: &expires,
		}).Error)
		for j, cost := range []int64{3000000, 9000000} {
			occurred := start.Add(time.Duration(j) * 2 * time.Hour)
			require.NoError(t, db.Create(&models.WorkspaceUsageEvent{
				ID: uuid.New(), OrganizationID: org.ID, CanvasRunID: uuid.New(),
				NodeExecutionID: uuid.New(), NodeID: "stats", Provider: "runner", UsageKind: "compute", FundingSource: "hosted",
				DurationSeconds: 60, MachineType: "e1-large-amd64", FleetID: "local", CostMicros: cost, Currency: "USD", IdempotencyKey: uuid.NewString(), OccurredAt: occurred, CreatedAt: occurred,
			}).Error)
		}
	}
	query := url.Values{"start": {start.Format(time.RFC3339)}, "end": {end.Format(time.RFC3339)}}
	request := func(offset string) map[string]json.RawMessage {
		query.Set("offset", offset)
		response := execRequest(server, requestParams{method: "GET", path: "/admin/api/stats?" + query.Encode(), authCookie: token})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var result map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		return result
	}
	first := request("0")
	require.JSONEq(t, `6`, string(first["total"]))
	require.JSONEq(t, `6000`, string(first["trial_granted_cents"]))
	// Unused expired credit must not be counted as spent. Only the usage before expiry consumed it.
	require.JSONEq(t, `2500`, string(first["trial_spent_cents"]))
	var rows []struct {
		ID         uuid.UUID
		OwnerEmail string `json:"owner_email"`
		Plan       string
		Available  int64 `json:"available_credit_cents"`
	}
	require.NoError(t, json.Unmarshal(first["items"], &rows))
	require.Len(t, rows, 5)
	require.Equal(t, ids[5], rows[0].ID)
	require.Equal(t, r.Account.Email, rows[0].OwnerEmail)
	require.Equal(t, "Trial", rows[0].Plan)
	require.EqualValues(t, 2800, rows[0].Available)
	require.Equal(t, "None", rows[1].Plan)
	require.Zero(t, rows[1].Available)
	beyond := request("10")
	require.JSONEq(t, `[]`, string(beyond["items"]))
	require.NotContains(t, beyond, "trial_granted_cents")
	query.Set("start", start.AddDate(-1, 0, 0).Format(time.RFC3339))
	query.Set("end", start.AddDate(-1, 0, 1).Format(time.RFC3339))
	empty := request("0")
	require.JSONEq(t, `0`, string(empty["total"]))
	require.JSONEq(t, `0`, string(empty["trial_granted_cents"]))
	require.JSONEq(t, `0`, string(empty["trial_spent_cents"]))
	query.Set("start", start.Format(time.RFC3339))
	query.Set("end", end.Format(time.RFC3339))
	second := request("5")
	require.JSONEq(t, `6`, string(second["total"]))
	require.NotContains(t, second, "trial_granted_cents")
	require.NotContains(t, second, "trial_spent_cents")
	require.NoError(t, json.Unmarshal(second["items"], &rows))
	require.Len(t, rows, 1)
	require.Equal(t, ids[0], rows[0].ID)
	for _, query := range []string{"", "?start=invalid&end=invalid", "?start=" + end.Format(time.RFC3339) + "&end=" + start.Format(time.RFC3339)} {
		response := execRequest(server, requestParams{method: "GET", path: "/admin/api/stats" + query, authCookie: token})
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
	response := execRequest(server, requestParams{method: "GET", path: "/admin/api/stats?" + query.Encode()})
	require.NotEqual(t, http.StatusOK, response.Code)
	require.NoError(t, models.DemoteFromInstallationAdmin(r.Account.ID.String()))
	response = execRequest(server, requestParams{method: "GET", path: "/admin/api/stats?" + query.Encode(), authCookie: token})
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestAdminRegistrationStatsPlanActivations(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	db := database.DB(context.WithoutCancel(t.Context()))
	windowStart := time.Now().UTC().AddDate(0, 0, -60).Truncate(time.Second)
	windowEnd := windowStart.AddDate(0, 0, 1)
	setPlanActivationTrackingStart(t, db, windowStart.AddDate(0, 0, -365))

	registeredInWindow := newStatsOrganization(t, db, windowStart.Add(time.Hour))
	registeredEarlier := newStatsOrganization(t, db, windowStart.AddDate(-1, 0, 0))
	insertPlanActivation(t, db, registeredEarlier, billingmodels.PlanActivationKeyTeam, windowStart)
	insertPlanActivation(t, db, registeredEarlier, billingmodels.PlanActivationKeyTeam, windowEnd.Add(-time.Microsecond))
	insertPlanActivation(t, db, registeredEarlier, billingmodels.PlanActivationKeyTeam, windowEnd)
	insertPlanActivation(t, db, registeredEarlier, billingmodels.PlanActivationKeyTeam, windowStart.Add(-time.Second))
	insertPlanActivation(t, db, registeredInWindow, billingmodels.PlanActivationKeyStarter, windowStart.Add(time.Hour))
	insertPlanActivation(t, db, registeredInWindow, billingmodels.PlanActivationKeyPayAsYouGo, windowStart.Add(2*time.Hour))
	insertPlanActivation(t, db, registeredInWindow, billingmodels.PlanActivationKeyPayAsYouGo, windowStart.Add(3*time.Hour))
	insertPlanActivation(t, db, registeredInWindow, "business_legacy", windowStart.Add(4*time.Hour))

	t.Run("counts activations by activation time with an inclusive start and exclusive end", func(t *testing.T) {
		for _, offset := range []string{"0", "1"} {
			report := getStats(t, server, token, windowStart, windowEnd, offset)
			assert.JSONEq(t, `{"team":2,"starter":1,"pay_as_you_go":2}`, string(report["plan_activations"]), "offset %s", offset)
			assert.JSONEq(t, `1`, string(report["total"]), "registration cohort is separate, offset %s", offset)
		}
	})

	t.Run("returns zero counts and the tracking start for a period without activations", func(t *testing.T) {
		start := windowStart.AddDate(0, 0, -10)
		report := getStats(t, server, token, start, start.AddDate(0, 0, 1), "0")
		assert.JSONEq(t, `{"team":0,"starter":0,"pay_as_you_go":0}`, string(report["plan_activations"]))
		assert.JSONEq(t, `0`, string(report["total"]))
		assert.JSONEq(t, `"`+windowStart.AddDate(0, 0, -365).Format(time.RFC3339Nano)+`"`, string(report["plan_activation_tracking_started_at"]))
	})

	t.Run("counts activations when no organization registered in the period", func(t *testing.T) {
		start := windowStart.AddDate(0, 0, -2)
		insertPlanActivation(t, db, registeredEarlier, billingmodels.PlanActivationKeyTeam, start.Add(12*time.Hour))
		report := getStats(t, server, token, start, start.AddDate(0, 0, 1), "0")
		assert.JSONEq(t, `{"team":1,"starter":0,"pay_as_you_go":0}`, string(report["plan_activations"]))
		assert.JSONEq(t, `0`, string(report["total"]))
		assert.JSONEq(t, `[]`, string(report["items"]))
		assert.JSONEq(t, `0`, string(report["trial_granted_cents"]))
	})

	t.Run("excludes activations before the tracking start", func(t *testing.T) {
		cutoff := windowStart.Add(12 * time.Hour)
		setPlanActivationTrackingStart(t, db, cutoff)
		insertPlanActivation(t, db, registeredInWindow, billingmodels.PlanActivationKeyStarter, cutoff.Add(time.Hour))
		insertPlanActivation(t, db, registeredInWindow, billingmodels.PlanActivationKeyPayAsYouGo, cutoff.Add(2*time.Hour))
		report := getStats(t, server, token, windowStart, windowEnd, "0")
		// Team at the window start and the starter and pay-as-you-go events
		// before the cutoff are excluded. The team event before the window end
		// stays. The events after the cutoff count.
		assert.JSONEq(t, `{"team":1,"starter":1,"pay_as_you_go":1}`, string(report["plan_activations"]))
		assert.JSONEq(t, `"`+cutoff.Format(time.RFC3339Nano)+`"`, string(report["plan_activation_tracking_started_at"]))
	})
}

func setPlanActivationTrackingStart(t *testing.T, db *gorm.DB, at time.Time) {
	t.Helper()
	var original time.Time
	require.NoError(t, db.Raw("SELECT started_at FROM plan_activation_tracking WHERE id = 1").Scan(&original).Error)
	require.NoError(t, db.Exec("UPDATE plan_activation_tracking SET started_at = ? WHERE id = 1", at.UTC()).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec("UPDATE plan_activation_tracking SET started_at = ? WHERE id = 1", original).Error)
	})
}

func newStatsOrganization(t *testing.T, db *gorm.DB, createdAt time.Time) models.Organization {
	t.Helper()
	org := models.Organization{ID: uuid.New(), Name: "Activation organization", Slug: uuid.NewString(), CreatedAt: &createdAt}
	require.NoError(t, db.Create(&org).Error)
	t.Cleanup(func() {
		cleanupDB := database.DB(context.WithoutCancel(t.Context()))
		require.NoError(t, cleanupDB.Where("organization_id = ?", org.ID).Delete(&billingmodels.PlanActivation{}).Error)
		require.NoError(t, cleanupDB.Unscoped().Where("id = ?", org.ID).Delete(&models.Organization{}).Error)
	})
	return org
}

func insertPlanActivation(t *testing.T, db *gorm.DB, org models.Organization, key string, at time.Time) {
	t.Helper()
	activation := billingmodels.NewPlanActivation(org.ID, key, at)
	require.NoError(t, db.Create(&activation).Error)
}

func getStats(t *testing.T, server *testServer, token string, start, end time.Time, offset string) map[string]json.RawMessage {
	t.Helper()
	query := url.Values{"start": {start.Format(time.RFC3339Nano)}, "end": {end.Format(time.RFC3339Nano)}, "offset": {offset}}
	response := execRequest(server, requestParams{method: "GET", path: "/admin/api/stats?" + query.Encode(), authCookie: token})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	return result
}
