package billinghttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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
	require.JSONEq(t, `6000`, string(beyond["trial_granted_cents"]))
	second := request("5")
	require.JSONEq(t, `6`, string(second["total"]))
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
