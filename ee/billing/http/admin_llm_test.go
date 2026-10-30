package billinghttp

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func setAdminBillingTestProduct(t *testing.T, productID string) {
	t.Helper()
	db := database.DB(t.Context())
	catalog, err := billingmodels.FindBillingPlan(db, billingmodels.BillingPlanBusiness)
	require.NoError(t, err)
	originalProduct := catalog.PolarProductID
	if originalProduct != nil {
		originalProduct = new(*originalProduct)
	}
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.WithoutCancel(t.Context())).Model(catalog).Update("polar_product_id", originalProduct).Error)
	})
	require.NoError(t, db.Model(catalog).Update("polar_product_id", productID).Error)
}

func TestAdminOrganizationCreditBalances(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	creditPath := "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-credit"
	balancesPath := creditPath + "/balances"
	setBalance := func(authCookie string, body map[string]any) *httptest.ResponseRecorder {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		return execRequest(server, requestParams{
			method:      "PUT",
			path:        balancesPath,
			authCookie:  authCookie,
			body:        encoded,
			contentType: "application/json",
		})
	}

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-credit-balance@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := setBalance(regularToken, map[string]any{"bucket": "grant", "target_cents": 100})
		assert.Equal(t, http.StatusNotFound, response.Code)
		response = execRequest(server, requestParams{method: "GET", path: creditPath + "/grants", authCookie: regularToken})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin sets trial balance and sees the adjustment in history", func(t *testing.T) {
		response := execRequest(server, requestParams{method: "GET", path: creditPath, authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)
		var credit organizationLLMCreditResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.Equal(t, billingmodels.SignupCreditCents, credit.WelcomeRemainingCents)
		assert.NotNil(t, credit.WelcomeCreditExpiresAt)

		response = setBalance(token, map[string]any{
			"bucket":                   "trial",
			"target_cents":             400,
			"expected_remaining_cents": credit.WelcomeRemainingCents,
			"note":                     "support",
		})
		require.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.Equal(t, int64(400), credit.WelcomeRemainingCents)
		assert.Equal(t, int64(400), credit.RemainingCreditCents)

		response = execRequest(server, requestParams{method: "GET", path: creditPath + "/grants", authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)
		var history struct {
			Grants []map[string]any `json:"grants"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &history))
		require.Len(t, history.Grants, 2)
		latest := history.Grants[0]
		assert.Equal(t, billingmodels.LLMCreditGrantKindTrialAdjustment, latest["kind"])
		assert.Equal(t, "-600", latest["amountCents"])
		assert.Equal(t, "support", latest["note"])
		assert.NotEmpty(t, latest["actorName"])
	})

	t.Run("stale expected balance returns 409", func(t *testing.T) {
		response := setBalance(token, map[string]any{
			"bucket":                   "trial",
			"target_cents":             2000,
			"expected_remaining_cents": 5000,
		})
		assert.Equal(t, http.StatusConflict, response.Code)
		assert.Contains(t, response.Body.String(), "The balance changed")
	})

	t.Run("negative target and unknown credit type return 400", func(t *testing.T) {
		response := setBalance(token, map[string]any{"bucket": "grant", "target_cents": -100})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		response = setBalance(token, map[string]any{"bucket": "included", "target_cents": 100})
		assert.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("target outside the safe cent range returns 400", func(t *testing.T) {
		response := setBalance(token, map[string]any{
			"bucket":                   "grant",
			"target_cents":             math.MaxInt64,
			"expected_remaining_cents": 0,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)

		response = setBalance(token, map[string]any{
			"bucket":                   "grant",
			"target_cents":             math.MinInt64,
			"expected_remaining_cents": 0,
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)

		response = execRequest(server, requestParams{method: "GET", path: creditPath, authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)
		var credit organizationLLMCreditResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.Equal(t, int64(0), credit.AdminRemainingCents)
	})
}

func TestAdminOrganizationBillingPlanHiddenWhenBillingDisabled(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"
	sync := execRequest(server, requestParams{method: http.MethodPost, path: "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-customer/sync", authCookie: token})
	assert.Equal(t, http.StatusNotFound, sync.Code)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		response := execRequest(server, requestParams{
			method:      method,
			path:        path,
			authCookie:  token,
			body:        []byte(`{"plan":"business"}`),
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusNotFound, response.Code, method)
		assert.Contains(t, response.Body.String(), "Billing is not available")
	}
}

func TestAdminCatalogPlanAssignment(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	catalog := billingmodels.BillingPlan{Key: "support_plan", Name: "Support plan", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	t.Cleanup(func() {
		_, err := billingmodels.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, "none", nil)
		require.NoError(t, err)
		require.NoError(t, billingmodels.DeleteBillingPlan(database.Conn(), catalog.ID))
	})

	regular, err := models.CreateAccount("Catalog user", "catalog-user@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	denied := execRequest(server, requestParams{method: "GET", path: "/admin/api/billing-plans", authCookie: regularToken})
	require.Equal(t, http.StatusNotFound, denied.Code)

	response := execRequest(server, requestParams{method: "GET", path: "/admin/api/billing-plans", authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	var plans []billingmodels.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plans))
	found := false
	for _, plan := range plans {
		if plan.Key == catalog.Key {
			found = true
			require.Equal(t, catalog.ID, plan.ID)
		}
	}
	require.True(t, found, "private catalog plan must be listed")

	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"
	previous, err := billingmodels.ResolveOrganizationBillingPlan(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	catalog.RequirePaymentMethod = true
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	response = execRequest(server, requestParams{method: "PUT", path: path, authCookie: token, body: []byte(`{"plan":"support_plan"}`), contentType: "application/json"})
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "Plans that require a payment method cannot be assigned by an admin")
	unchanged, err := billingmodels.ResolveOrganizationBillingPlan(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, previous.PlanID, unchanged.PlanID)
	require.Equal(t, previous.PlanSource, unchanged.PlanSource)

	catalog.RequirePaymentMethod = false
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	response = execRequest(server, requestParams{method: "PUT", path: path, authCookie: token, body: []byte(`{"plan":"support_plan"}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code)
	var saved map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &saved))
	require.Equal(t, catalog.Key, saved["catalog_key"])
	assigned, err := billingmodels.ResolveOrganizationBillingPlan(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, catalog.ID, assigned.PlanID)

	response = execRequest(server, requestParams{method: "GET", path: path, authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &saved))
	require.Equal(t, catalog.Key, saved["catalog_key"])
}

func TestAdminOrganizationBillingPlan(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"
	polarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer polarServer.Close()
	t.Setenv("POLAR_API_BASE_URL", polarServer.URL)

	t.Run("admin can set a plan when Polar is not set up", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{"plan": "business"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var plan organizationBillingPlanResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
		assert.Equal(t, billingmodels.BillingPlanBusiness, plan.Plan)
		assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
		assert.False(t, plan.PolarManaged)
	})

	t.Run("PUT allows a plan when only a Polar customer exists", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.Conn(), r.Organization.ID, "cust_polar"))

		body, err := json.Marshal(map[string]any{
			"plan":          "trial",
			"trial_ends_at": time.Now().Add(20 * 24 * time.Hour).UTC().Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var plan organizationBillingPlanResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
		assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
		assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
		assert.False(t, plan.PolarManaged)
	})

	t.Run("GET verifies Polar paid without changing the admin plan", func(t *testing.T) {
		periodStart := time.Now().UTC().Truncate(time.Second)
		periodEnd := periodStart.AddDate(0, 1, 0)
		polarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/subscriptions/" {
				http.NotFound(w, req)
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":                   "sub_admin_sync",
						"product_id":           "prod_business",
						"status":               "active",
						"current_period_start": periodStart.Format(time.RFC3339),
						"current_period_end":   periodEnd.Format(time.RFC3339),
						"customer_id":          "cust_polar_1",
						"external_customer_id": r.Organization.ID.String(),
						"customer": map[string]any{
							"id":          "cust_polar_1",
							"external_id": r.Organization.ID.String(),
						},
					},
				},
				"pagination": map[string]any{"max_page": 1},
			}))
		}))
		t.Cleanup(polarServer.Close)
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
		t.Setenv("POLAR_API_BASE_URL", polarServer.URL)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       path + "?check_polar=true",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var plan organizationBillingPlanResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
		assert.Equal(t, billingmodels.BillingPlanTrial, plan.Plan)
		assert.Equal(t, billingmodels.BillingPlanSourceAdmin, plan.PlanSource)
		assert.Equal(t, "active", plan.PolarSubscriptionVerification)
	})

	t.Run("PUT rejects Polar-managed organizations", func(t *testing.T) {
		activeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"items":[{"id":"sub_admin_put","status":"active"}]}`))
		}))
		defer activeServer.Close()
		t.Setenv("POLAR_API_BASE_URL", activeServer.URL)
		setAdminBillingTestProduct(t, "prod_business")
		now := time.Now()
		end := now.AddDate(0, 1, 0)
		_, err := billingmodels.ApplyPolarSubscription(database.Conn(), r.Organization.ID, billingmodels.PolarSubscriptionApply{ProductID: "prod_business",
			ID:          "sub_admin_put",
			Status:      billingmodels.PolarSubscriptionStatusActive,
			PeriodStart: &now,
			PeriodEnd:   &end,
		}, nil)
		require.NoError(t, err)
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")

		body, err := json.Marshal(map[string]any{
			"plan":          "trial",
			"trial_ends_at": time.Now().Add(20 * 24 * time.Hour).UTC().Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Cancel or change the subscription in Polar")
	})
}

func TestAdminOrganizationBillingPlanTrialEnd(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"

	t.Run("PUT trial rejects a missing end", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{"plan": "trial"})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "trial end date is required")
	})

	t.Run("PUT trial rejects an end that is not in the future", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"plan":          "trial",
			"trial_ends_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "choose a future date")
	})

	t.Run("PUT trial stores the end and reopens expired welcome credit", func(t *testing.T) {
		ended := time.Now().Add(-time.Hour)
		require.NoError(t, database.Conn().Model(&billingmodels.OrganizationLLMCreditGrant{}).
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			Update("expires_at", ended).Error)
		var before billingmodels.OrganizationLLMCreditGrant
		require.NoError(t, database.Conn().
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			First(&before).Error)

		trialEnd := time.Now().Add(18 * 24 * time.Hour).UTC().Truncate(time.Second)
		body, err := json.Marshal(map[string]any{
			"plan":          "trial",
			"trial_ends_at": trialEnd.Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var plan organizationBillingPlanResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
		require.NotNil(t, plan.TrialEndsAt)
		parsed, err := time.Parse(time.RFC3339, *plan.TrialEndsAt)
		require.NoError(t, err)
		assert.True(t, parsed.Equal(trialEnd))

		var welcome billingmodels.OrganizationLLMCreditGrant
		require.NoError(t, database.Conn().
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			First(&welcome).Error)
		require.NotNil(t, welcome.ExpiresAt)
		assert.True(t, welcome.ExpiresAt.Equal(trialEnd))
		assert.Equal(t, before.AmountMicros, welcome.AmountMicros)

		summary, err := billingmodels.DescribeOrganizationLLMCredit(database.Conn(), r.Organization.ID)
		require.NoError(t, err)
		assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)
	})

	t.Run("PUT business and none ignore trial end", func(t *testing.T) {
		var before billingmodels.OrganizationLLMCreditGrant
		require.NoError(t, database.Conn().
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			First(&before).Error)
		ignored := time.Now().Add(90 * 24 * time.Hour).UTC().Format(time.RFC3339)

		for _, planName := range []string{"business", "none"} {
			body, err := json.Marshal(map[string]any{
				"plan":          planName,
				"trial_ends_at": ignored,
			})
			require.NoError(t, err)
			response := execRequest(server, requestParams{
				method:      "PUT",
				path:        path,
				authCookie:  token,
				body:        body,
				contentType: "application/json",
			})
			assert.Equal(t, http.StatusOK, response.Code, planName)

			var welcome billingmodels.OrganizationLLMCreditGrant
			require.NoError(t, database.Conn().
				Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
				First(&welcome).Error)
			require.NotNil(t, welcome.ExpiresAt)
			require.NotNil(t, before.ExpiresAt)
			assert.True(t, welcome.ExpiresAt.Equal(*before.ExpiresAt), planName)
		}
	})
}

func TestAdminOrganizationBillingPlanCustomTrialEnd(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	catalog := billingmodels.BillingPlan{
		Key:              "custom_trial",
		Name:             "Custom trial",
		Category:         "trial",
		Visibility:       "private",
		CreditFrequency:  "none",
		CreditExpiration: "never",
		TrialLengthDays:  30,
	}
	require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &catalog))
	t.Cleanup(func() {
		_, err := billingmodels.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, "none", nil)
		require.NoError(t, err)
		require.NoError(t, billingmodels.DeleteBillingPlan(database.Conn(), catalog.ID))
	})
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"

	t.Run("PUT rejects a missing end", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        []byte(`{"plan":"custom_trial"}`),
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "trial end date is required")
	})

	t.Run("PUT rejects an end that is not in the future", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"plan":          "custom_trial",
			"trial_ends_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "choose a future date")
	})

	t.Run("PUT stores the end and reopens expired welcome credit", func(t *testing.T) {
		ended := time.Now().Add(-time.Hour)
		require.NoError(t, database.Conn().Model(&billingmodels.OrganizationLLMCreditGrant{}).
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			Update("expires_at", ended).Error)

		trialEnd := time.Now().Add(18 * 24 * time.Hour).UTC().Truncate(time.Second)
		body, err := json.Marshal(map[string]any{
			"plan":          "custom_trial",
			"trial_ends_at": trialEnd.Format(time.RFC3339),
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PUT",
			path:        path,
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var plan organizationBillingPlanResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
		assert.Equal(t, "custom_trial", plan.Plan)
		assert.Equal(t, "custom_trial", plan.CatalogKey)
		assert.Equal(t, "trial", plan.CatalogCategory)
		require.NotNil(t, plan.TrialEndsAt)
		parsed, err := time.Parse(time.RFC3339, *plan.TrialEndsAt)
		require.NoError(t, err)
		assert.True(t, parsed.Equal(trialEnd))

		var welcome billingmodels.OrganizationLLMCreditGrant
		require.NoError(t, database.Conn().
			Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).
			First(&welcome).Error)
		require.NotNil(t, welcome.ExpiresAt)
		assert.True(t, welcome.ExpiresAt.Equal(trialEnd))

		summary, err := billingmodels.DescribeOrganizationLLMCredit(database.Conn(), r.Organization.ID)
		require.NoError(t, err)
		assert.Equal(t, welcome.AmountMicros, summary.WelcomeRemainingMicros)
	})
}

func TestAdminPlanPolarVerification(t *testing.T) {
	for _, tc := range []struct {
		name         string
		items        []map[string]any
		raw          string
		code         int
		verification string
		saveCode     int
		concurrent   bool
		paginate     bool
		storedStatus string
	}{
		{name: "stale active", items: []map[string]any{}, verification: "none", saveCode: 200},
		{name: "ended", items: []map[string]any{{"id": "sub_ended", "status": "active", "ended_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}}, verification: "none", saveCode: 200},
		{name: "ended with cancel flag", items: []map[string]any{{"id": "sub_ended", "status": "canceled", "ended_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), "cancel_at_period_end": true, "current_period_end": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}}, verification: "none", saveCode: 200},
		{name: "active unknown product", items: []map[string]any{{"id": "sub_unknown", "product_id": "unknown", "status": "active"}}, verification: "active", saveCode: 400},
		{name: "trialing", items: []map[string]any{{"id": "sub_trial", "status": "trialing"}}, verification: "active", saveCode: 400},
		{name: "paused without scheduled cancellation", items: []map[string]any{{"id": "sub_stored", "status": "paused", "cancel_at_period_end": false}}, verification: "active", saveCode: 400, storedStatus: "paused"},
		{name: "ended paused", items: []map[string]any{{"id": "sub_stored", "status": "paused", "ended_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}}, verification: "none", saveCode: 200, storedStatus: "paused"},
		{name: "canceling", items: []map[string]any{{"id": "sub_canceling", "status": "canceled", "cancel_at_period_end": true, "current_period_end": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}}, verification: "active", saveCode: 400},
		{name: "canceled", items: []map[string]any{{"id": "sub_canceled", "status": "canceled"}}, verification: "none", saveCode: 200},
		{name: "pagination", items: []map[string]any{{"id": "sub_page2", "status": "active"}}, verification: "active", saveCode: 400, paginate: true},
		{name: "unauthorized", code: 401, verification: "failed", saveCode: 502},
		{name: "upstream timeout", code: 504, verification: "failed", saveCode: 502},
		{name: "unexpected response", raw: `{}`, verification: "failed", saveCode: 502},
		{name: "null items", raw: `{"items":null}`, verification: "failed", saveCode: 502},
		{name: "missing id", raw: `{"items":[{"status":"active"}]}`, verification: "failed", saveCode: 502},
		{name: "missing status", raw: `{"items":[{"id":"sub_unknown"}]}`, verification: "failed", saveCode: 502},
		{name: "wrong customer", items: []map[string]any{{"id": "sub_other", "status": "active", "external_customer_id": "another-organization"}}, verification: "failed", saveCode: 502},
		{name: "unknown status", items: []map[string]any{{"id": "sub_unknown", "status": "surprise"}}, verification: "failed", saveCode: 502},
		{name: "concurrent update", items: []map[string]any{}, verification: "none", saveCode: 409, concurrent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, r, token := setupAdminTestServer(t)
			now := time.Now()
			end := now.Add(time.Hour)
			setAdminBillingTestProduct(t, "prod_verify")
			catalog, err := billingmodels.FindBillingPlan(database.Conn(), billingmodels.BillingPlanBusiness)
			require.NoError(t, err)
			require.NotNil(t, catalog.PolarProductID)
			storedStatus := tc.storedStatus
			if storedStatus == "" {
				storedStatus = "active"
			}
			_, err = billingmodels.ApplyPolarSubscription(database.Conn(), r.Organization.ID, billingmodels.PolarSubscriptionApply{ID: "sub_stored", ProductID: *catalog.PolarProductID, Status: "active", PeriodStart: &now, PeriodEnd: &end, CancelAtPeriodEnd: storedStatus == "active"}, nil)
			require.NoError(t, err)
			if storedStatus == "paused" {
				require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).
					Where("organization_id = ?", r.Organization.ID).Update("trial_ends_at", now.Add(-time.Hour)).Error)
				_, err = billingmodels.ApplyPolarSubscription(database.Conn(), r.Organization.ID, billingmodels.PolarSubscriptionApply{ID: "sub_stored", ProductID: *catalog.PolarProductID, Status: storedStatus, PeriodStart: &now, PeriodEnd: &end}, nil)
				require.NoError(t, err)
			}
			saving := false
			calls := 0
			polarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/subscriptions/" {
					http.NotFound(w, req)
					return
				}
				calls++
				assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
				assert.Empty(t, req.URL.Query().Get("product_id"))
				if saving && tc.concurrent {
					require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("updated_at", time.Now().Add(time.Second)).Error)
				}
				if tc.code != 0 {
					w.WriteHeader(tc.code)
					return
				}
				if tc.raw != "" {
					_, err := w.Write([]byte(tc.raw))
					require.NoError(t, err)
					return
				}
				items := tc.items
				maxPage := 1
				if tc.paginate {
					maxPage = 2
				}
				if tc.paginate && req.URL.Query().Get("page") == "1" {
					items = []map[string]any{{"id": "sub_old", "status": "canceled"}}
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]any{"max_page": maxPage}}))
			}))
			defer polarServer.Close()
			t.Setenv("POLAR_ACCESS_TOKEN", "test")
			t.Setenv("POLAR_API_BASE_URL", polarServer.URL)
			t.Setenv("POLAR_ENVIRONMENT", "production")
			t.Setenv("POLAR_DASHBOARD_ORGANIZATION_SLUG", "superplane-inc")
			path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"
			get := func() organizationBillingPlanResponse {
				response := execRequest(server, requestParams{method: "GET", path: path + "?check_polar=true", authCookie: token})
				require.Equal(t, 200, response.Code, response.Body.String())
				var plan organizationBillingPlanResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
				return plan
			}
			plan := get()
			assert.Equal(t, tc.verification, plan.PolarSubscriptionVerification)
			assert.Equal(t, "sub_stored", plan.PolarSubscriptionID)
			assert.Equal(t, "https://polar.sh/dashboard/superplane-inc/sales/subscriptions/sub_stored", plan.PolarSubscriptionURL)
			assert.Equal(t, "polar", plan.PlanSource)
			assert.Equal(t, storedStatus, plan.PolarSubscriptionStatus)
			beforeSave := calls
			saving = true
			response := execRequest(server, requestParams{method: "PUT", path: path, authCookie: token, body: []byte(`{"plan":"business_manual"}`), contentType: "application/json"})
			require.Equal(t, tc.saveCode, response.Code, response.Body.String())
			assert.Greater(t, calls, beforeSave)
			saving = false
			after := get()
			if tc.saveCode == 200 {
				assert.Equal(t, "business_manual", after.CatalogKey)
				assert.Equal(t, "admin", after.PlanSource)
				assert.Empty(t, after.PolarSubscriptionStatus)
				assert.False(t, after.PolarManaged)
				assert.Equal(t, "none", after.PolarSubscriptionVerification)
				stored, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), r.Organization.ID)
				require.NoError(t, err)
				assert.False(t, stored.CancelAtPeriodEnd)
				assert.Equal(t, "sub_stored", after.PolarSubscriptionID)
			} else {
				assert.Equal(t, "polar", after.PlanSource)
				assert.Equal(t, "sub_stored", after.PolarSubscriptionID)
				assert.Equal(t, storedStatus, after.PolarSubscriptionStatus)
				if storedStatus == "paused" {
					stored, err := billingmodels.FindOrganizationBillingPlan(database.DB(t.Context()), r.Organization.ID)
					require.NoError(t, err)
					assert.False(t, stored.CancelAtPeriodEnd)
					assert.False(t, stored.IsActiveBusiness())
					assert.False(t, stored.AllowsHostedExecution(time.Now()))
					assert.False(t, stored.AllowsCreditPurchase())
				}
			}
		})
	}
}

func TestAdminLLMSettings(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	clearHostedProviderDefaults(t)
	original, err := billingmodels.GetInstallationLLMSettings(database.Conn())
	require.NoError(t, err)
	_, err = billingmodels.UpdateInstallationLLMSettings(database.Conn(), billingmodels.InstallationLLMSettings{
		WelcomeGrantCents:   billingmodels.DefaultWelcomeGrantCents,
		MarkupBPS:           billingmodels.DefaultMarkupBPS,
		WarningThresholdBPS: billingmodels.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, restoreErr := billingmodels.UpdateInstallationLLMSettings(database.Conn(), *original)
		require.NoError(t, restoreErr)
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-llm@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/llm-settings",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-settings",
			authCookie:  regularToken,
			body:        []byte(`{"credit_purchase_fee_bps":500}`),
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("admin can load defaults and update markup", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/llm-settings",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var settings installationLLMSettingsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		assert.Equal(t, billingmodels.DefaultWelcomeGrantCents, settings.WelcomeGrantCents)
		assert.Equal(t, billingmodels.DefaultMarkupBPS, settings.MarkupBPS)
		assert.Zero(t, settings.CreditPurchaseFeeBPS)
		assert.Zero(t, settings.CreditPurchaseFixedFeeCents)
		assert.Empty(t, settings.DefaultHostedProvider)
		assert.Empty(t, settings.DefaultHostedModel)
		require.Len(t, settings.Providers, 3)

		body, err := json.Marshal(map[string]any{
			"markup_bps": 1000,
		})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-settings",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		assert.Equal(t, billingmodels.DefaultWelcomeGrantCents, settings.WelcomeGrantCents)
		assert.Equal(t, 1000, settings.MarkupBPS)
	})

	t.Run("credit fees persist through unrelated updates and allow explicit zero", func(t *testing.T) {
		for _, tc := range []struct {
			body  string
			bps   int
			fixed int64
		}{
			{`{"credit_purchase_fee_bps":500,"credit_purchase_fixed_fee_cents":50}`, 500, 50},
			{`{"warning_threshold_bps":2500}`, 500, 50},
			{`{"credit_purchase_fee_bps":0,"credit_purchase_fixed_fee_cents":0}`, 0, 0},
		} {
			response := execRequest(server, requestParams{
				method:      "PATCH",
				path:        "/admin/api/installation/llm-settings",
				authCookie:  token,
				body:        []byte(tc.body),
				contentType: "application/json",
			})
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var settings installationLLMSettingsResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
			assert.Equal(t, tc.bps, settings.CreditPurchaseFeeBPS)
			assert.Equal(t, tc.fixed, settings.CreditPurchaseFixedFeeCents)
			saved, err := billingmodels.GetInstallationLLMSettings(database.Conn())
			require.NoError(t, err)
			assert.Equal(t, tc.bps, saved.CreditPurchaseFeeBPS)
			assert.Equal(t, tc.fixed, saved.CreditPurchaseFixedFeeCents)
		}
	})

	t.Run("invalid credit fees return a client error", func(t *testing.T) {
		for _, body := range []string{
			`{"credit_purchase_fee_bps":-1}`,
			`{"credit_purchase_fee_bps":10000}`,
			`{"credit_purchase_fixed_fee_cents":-1}`,
			`{"credit_purchase_fee_bps":500.5}`,
		} {
			response := execRequest(server, requestParams{
				method:      "PATCH",
				path:        "/admin/api/installation/llm-settings",
				authCookie:  token,
				body:        []byte(body),
				contentType: "application/json",
			})
			assert.Equal(t, http.StatusBadRequest, response.Code, body)
		}
	})

	t.Run("admin can save a hosted provider", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"enabled":        true,
			"api_key":        "sk-test",
			"allowed_models": []string{"claude-sonnet-4-6"},
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-providers/anthropic",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var settings installationLLMSettingsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		var anthropic hostedLLMProviderResponse
		for _, provider := range settings.Providers {
			if provider.Provider == "anthropic" {
				anthropic = provider
			}
		}
		assert.True(t, anthropic.Enabled)
		assert.True(t, anthropic.APIKeyConfigured)
		assert.False(t, anthropic.ManagementKeyConfigured)
		assert.Equal(t, []string{"claude-sonnet-4-6"}, anthropic.AllowedModels)

		body, err = json.Marshal(map[string]any{
			"default_hosted_provider": "anthropic",
			"default_hosted_model":    "claude-sonnet-4-6",
		})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-settings",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		assert.Equal(t, "anthropic", settings.DefaultHostedProvider)
		assert.Equal(t, "claude-sonnet-4-6", settings.DefaultHostedModel)
	})

	t.Run("admin cannot enable OpenRouter without a provisioning key", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"enabled":        true,
			"api_key":        "sk-or-inference",
			"allowed_models": []string{"anthropic/claude-sonnet-4-6"},
		})
		require.NoError(t, err)
		response := execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/installation/llm-providers/openrouter",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "provisioning API key is required")
	})

	t.Run("admin GET reports when a provisioning key is stored", func(t *testing.T) {
		_, err := models.UpsertHostedLLMProvider(database.Conn(), models.HostedLLMProvider{
			Provider:      models.UsageProviderOpenRouter,
			Enabled:       false,
			APIKey:        []byte("encrypted-inference"),
			ManagementKey: []byte("encrypted-mgmt"),
			AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet-4-6"},
		})
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/installation/llm-settings",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var settings installationLLMSettingsResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &settings))
		var openrouterProvider hostedLLMProviderResponse
		for _, provider := range settings.Providers {
			if provider.Provider == "openrouter" {
				openrouterProvider = provider
			}
		}
		assert.True(t, openrouterProvider.APIKeyConfigured)
		assert.True(t, openrouterProvider.ManagementKeyConfigured)
		assert.NotContains(t, response.Body.String(), "encrypted-mgmt")
	})

	t.Run("admin can grant credit and set markup override", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-credit",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var credit organizationLLMCreditResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.Greater(t, credit.RemainingCreditCents, int64(0))
		assert.Equal(t, billingmodels.SignupCreditCents, credit.SuperplaneGrantCents)
		assert.Equal(t, int64(0), credit.PurchasedCreditCents)

		body, err := json.Marshal(map[string]any{
			"amount_cents": 1000,
			"note":         "restore",
		})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "POST",
			path:        "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-credit/grants",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		assert.GreaterOrEqual(t, credit.GrantTotalCents, billingmodels.SignupCreditCents+1000)
		assert.GreaterOrEqual(t, credit.SuperplaneGrantCents, billingmodels.SignupCreditCents)
		assert.Equal(t, int64(0), credit.PurchasedCreditCents)

		body, err = json.Marshal(map[string]any{"markup_bps": 0})
		require.NoError(t, err)
		response = execRequest(server, requestParams{
			method:      "PATCH",
			path:        "/admin/api/organizations/" + r.Organization.ID.String() + "/llm-settings",
			authCookie:  token,
			body:        body,
			contentType: "application/json",
		})
		assert.Equal(t, http.StatusOK, response.Code)
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &credit))
		require.NotNil(t, credit.MarkupOverrideBPS)
		assert.Equal(t, 0, *credit.MarkupOverrideBPS)
	})
}

type installationLLMSettingsResponse struct {
	WelcomeGrantCents           int64                       `json:"welcome_grant_cents"`
	MarkupBPS                   int                         `json:"markup_bps"`
	WarningThresholdBPS         int                         `json:"warning_threshold_bps"`
	CreditPurchaseFeeBPS        int                         `json:"credit_purchase_fee_bps"`
	CreditPurchaseFixedFeeCents int64                       `json:"credit_purchase_fixed_fee_cents"`
	DefaultHostedProvider       string                      `json:"default_hosted_provider"`
	DefaultHostedModel          string                      `json:"default_hosted_model"`
	Providers                   []hostedLLMProviderResponse `json:"providers"`
}

type hostedLLMProviderResponse struct {
	Provider                string   `json:"provider"`
	Enabled                 bool     `json:"enabled"`
	APIKeyConfigured        bool     `json:"api_key_configured"`
	ManagementKeyConfigured bool     `json:"management_key_configured"`
	BaseURL                 string   `json:"base_url"`
	AllowedModels           []string `json:"allowed_models"`
}
