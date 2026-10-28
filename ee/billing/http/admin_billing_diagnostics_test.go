package billinghttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// The admin read must preserve missing and expired records and never contact
// Polar until explicitly requested. Existing plan tests cover PUT, not this read.
func TestAdminBillingDiagnosticsReadOnly(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.DB(t.Context())
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		http.Error(w, "unexpected Polar request", 500)
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "test")
	t.Setenv("POLAR_API_BASE_URL", upstream.URL)
	path := "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan"
	read := func() map[string]any {
		response := execRequest(server, requestParams{method: "GET", path: path, authCookie: token})
		require.Equal(t, 200, response.Code, response.Body.String())
		var result map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		return result
	}
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
	result := read()
	assert.Equal(t, false, result["record_exists"])
	assert.Equal(t, "none", result["plan"])
	assert.Equal(t, "not_checked", result["polar_subscription_verification"])
	missing, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, missing)
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, "trial", nil)
	require.NoError(t, err)
	expired := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("trial_ends_at", expired).Error)
	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	result = read()
	assert.Equal(t, true, result["record_exists"])
	assert.Equal(t, "trial", result["plan"])
	after, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	assert.Zero(t, calls.Load())
}

func TestAdminBillingDiagnosticsManualCheck(t *testing.T) {
	for _, tc := range []struct {
		name, storedID, customerID, customerStatus, cardStatus string
		customerCode, cardCode, subscriptionCode               int
		expired, card, malformedCard                           bool
	}{
		{name: "missing mapping", customerID: "cust_live", customerStatus: "found", cardStatus: "usable", card: true},
		{name: "mismatched mapping", storedID: "cust_old", customerID: "cust_live", customerStatus: "found", cardStatus: "usable", card: true},
		{name: "expired card", customerID: "cust_live", customerStatus: "found", cardStatus: "absent", card: true, expired: true},
		{name: "absent card", customerID: "cust_live", customerStatus: "found", cardStatus: "absent"},
		{name: "missing customer", storedID: "cust_old", customerCode: 404, customerStatus: "absent", cardStatus: "absent"},
		{name: "customer failure", customerCode: 503, customerStatus: "failed", cardStatus: "unknown"},
		{name: "card failure", customerID: "cust_live", cardCode: 503, customerStatus: "found", cardStatus: "failed"},
		{name: "card endpoint missing", customerID: "cust_live", cardCode: 404, customerStatus: "found", cardStatus: "failed"},
		{name: "malformed cards", customerID: "cust_live", malformedCard: true, customerStatus: "found", cardStatus: "failed"},
		{name: "subscription failure", customerID: "cust_live", subscriptionCode: 503, customerStatus: "found", cardStatus: "usable", card: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, r, token := setupAdminTestServer(t)
			db := database.DB(t.Context())
			orgID := r.Organization.ID
			if tc.storedID != "" {
				require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(db, orgID, tc.storedID))
			}
			before, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
			require.NoError(t, err)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Polar-Version", req.Header.Get("Polar-Version"))
				assert.Equal(t, "GET", req.Method)
				switch req.URL.Path {
				case "/customers/external/" + orgID.String():
					if tc.customerCode != 0 {
						w.WriteHeader(tc.customerCode)
						return
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": tc.customerID, "external_id": orgID.String()}))
				case "/customers/external/" + orgID.String() + "/payment-methods":
					if tc.cardCode != 0 {
						w.WriteHeader(tc.cardCode)
						return
					}
					if tc.malformedCard {
						_, _ = w.Write([]byte(`{}`))
						return
					}
					items := []map[string]any{}
					if tc.card {
						expiry := time.Now().AddDate(1, 0, 0)
						if tc.expired {
							expiry = time.Now().AddDate(-1, 0, 0)
						}
						items = append(items, map[string]any{"id": "card", "type": "card", "method_metadata": map[string]any{"exp_month": int(expiry.Month()), "exp_year": expiry.Year()}})
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}}))
				case "/subscriptions/":
					assert.Equal(t, orgID.String(), req.URL.Query().Get("external_customer_id"))
					if tc.subscriptionCode != 0 {
						w.WriteHeader(tc.subscriptionCode)
						return
					}
					_, _ = w.Write([]byte(`{"items":[{"id":"sub_live","status":"paused"},{"id":"sub_old","status":"canceled"}]}`))
				default:
					t.Errorf("unexpected request: %s", req.URL)
					http.NotFound(w, req)
				}
			}))
			t.Cleanup(upstream.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "test")
			t.Setenv("POLAR_API_BASE_URL", upstream.URL)
			t.Setenv("POLAR_ENVIRONMENT", "sandbox")
			t.Setenv("POLAR_DASHBOARD_ORGANIZATION_SLUG", "superplane")
			start := time.Now().Add(-time.Second)
			response := execRequest(server, requestParams{method: "GET", path: "/admin/api/organizations/" + orgID.String() + "/billing-plan?check_polar=true", authCookie: token})
			require.Equal(t, 200, response.Code, response.Body.String())
			var result organizationBillingPlanResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, tc.customerStatus, result.Polar.CustomerStatus)
			assert.Equal(t, tc.cardStatus, result.Polar.CardStatus)
			assert.Equal(t, tc.storedID, result.PolarCustomerID)
			assert.Equal(t, tc.storedID != "" && tc.customerID != "" && tc.storedID != tc.customerID, result.Polar.CustomerIDMismatch)
			assert.Equal(t, "sandbox", result.Polar.Environment)
			require.NotNil(t, result.Polar.CheckedAt)
			assert.True(t, result.Polar.CheckedAt.After(start))
			if tc.subscriptionCode == 0 {
				assert.Equal(t, "active", result.PolarSubscriptionVerification)
				require.Len(t, result.Polar.Subscriptions, 2)
				assert.Equal(t, "paused", result.Polar.Subscriptions[0].Status)
			} else {
				assert.Equal(t, "failed", result.PolarSubscriptionVerification)
				assert.NotEmpty(t, result.Polar.SubscriptionError)
			}
			after, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
			require.NoError(t, err)
			assert.Equal(t, before, after)
			settings, err := billingmodels.FindOrganizationLLMSettings(db, orgID)
			require.NoError(t, err)
			if tc.storedID == "" {
				assert.True(t, settings == nil || settings.PolarCustomerID == nil)
			} else {
				require.NotNil(t, settings)
				assert.Equal(t, tc.storedID, *settings.PolarCustomerID)
			}
		})
	}
}

func TestAdminBillingDiagnosticsRecordedEvidence(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.DB(t.Context())
	now := time.Now().UTC().Truncate(time.Second)
	factory, err := models.CreateFactory(db, r.Organization.ID, "Completed workspace", "", "DONE")
	require.NoError(t, err)
	require.NoError(t, db.Model(factory).Update("onboarding_completed_at", now).Error)
	pending, err := models.CreateFactory(db, r.Organization.ID, "Pending workspace", "", "WAIT")
	require.NoError(t, err)
	for i := 0; i < 22; i++ {
		attempt := billingmodels.OrganizationBillingCheckout{ID: uuid.New(), OrganizationID: r.Organization.ID, PlanKey: "business", State: "failed", LastError: diagnosticStringPointer("Payment declined"), CreatedAt: now.Add(time.Duration(i) * time.Minute), UpdatedAt: now, ResolvedAt: &now, PolarStatus: diagnosticStringPointer("failed"), PolarCheckoutID: diagnosticStringPointer(uuid.NewString())}
		require.NoError(t, db.Create(&attempt).Error)
	}
	response := execRequest(server, requestParams{method: "GET", path: "/admin/api/organizations/" + r.Organization.ID.String() + "/billing-plan", authCookie: token})
	require.Equal(t, 200, response.Code, response.Body.String())
	var result organizationBillingPlanResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	require.Len(t, result.CheckoutAttempts, 20)
	assert.Equal(t, now.Add(21*time.Minute), result.CheckoutAttempts[0].CreatedAt)
	assert.Equal(t, now.Add(2*time.Minute), result.CheckoutAttempts[19].CreatedAt)
	assert.Equal(t, "Payment declined", *result.CheckoutAttempts[0].LastError)
	completedFound, pendingFound := false, false
	for _, workspace := range result.Workspaces {
		if workspace.ID == factory.ID {
			assert.Equal(t, &now, workspace.OnboardingCompletedAt)
			completedFound = true
		}
		if workspace.ID == pending.ID {
			assert.Nil(t, workspace.OnboardingCompletedAt)
			pendingFound = true
		}
	}
	assert.True(t, completedFound)
	assert.True(t, pendingFound)
}

func diagnosticStringPointer(value string) *string { return &value }
