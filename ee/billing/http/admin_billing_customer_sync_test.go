package billinghttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	billingorganizations "github.com/superplanehq/superplane/ee/billing/organizations"
	"github.com/superplanehq/superplane/pkg/database"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func TestAdminBillingCustomerSync(t *testing.T) {
	for _, tc := range []struct {
		name, storedID                                   string
		status                                           int
		customerCode, cardCode, subscriptionCode         int
		noCard, expired                                  bool
		foreignCustomer, emptyCustomer                   bool
		missingPlan, notConfigured, precheck, noSettings bool
	}{
		{name: "missing mapping and plan", missingPlan: true, noSettings: true, status: 200},
		{name: "incorrect mapping and expired trial", storedID: "cust_old", status: 200},
		{name: "current mapping", storedID: "cust_live", status: 200},
		{name: "subscription failure", storedID: "cust_old", subscriptionCode: 503, status: 200},
		{name: "missing customer", storedID: "cust_old", customerCode: 404, status: 422},
		{name: "customer lookup failure", storedID: "cust_old", customerCode: 503, status: 502},
		{name: "wrong external id", storedID: "cust_old", foreignCustomer: true, status: 502},
		{name: "empty customer id", storedID: "cust_old", emptyCustomer: true, status: 502},
		{name: "no card", storedID: "cust_old", noCard: true, status: 422},
		{name: "expired card", storedID: "cust_old", expired: true, status: 422},
		{name: "card lookup failure", storedID: "cust_old", cardCode: 503, status: 502},
		{name: "card endpoint missing", storedID: "cust_old", cardCode: 404, status: 502},
		{name: "card removed after check", storedID: "cust_old", precheck: true, status: 422},
		{name: "not configured", storedID: "cust_old", notConfigured: true, status: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, r, token := setupAdminTestServer(t)
			db := database.DB(t.Context())
			orgID := r.Organization.ID
			markup := 1700
			if !tc.noSettings {
				require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, orgID, &markup))
			}
			if tc.storedID != "" {
				require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(db, orgID, tc.storedID))
			}
			if tc.missingPlan {
				require.NoError(t, db.Where("organization_id = ?", orgID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
			} else {
				_, err := billingmodels.SetAdminOrganizationPlan(db, orgID, "trial", nil)
				require.NoError(t, err)
				require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).Update("trial_ends_at", time.Now().Add(-time.Hour)).Error)
			}
			previousPlan, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
			require.NoError(t, err)
			previousSettings, err := billingmodels.FindOrganizationLLMSettings(db, orgID)
			require.NoError(t, err)
			var requests atomic.Int32
			var removedCard atomic.Bool
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests.Add(1)
				w.Header().Set("Polar-Version", req.Header.Get("Polar-Version"))
				assert.Equal(t, "GET", req.Method)
				switch req.URL.Path {
				case "/customers/external/" + orgID.String():
					if tc.customerCode != 0 {
						w.WriteHeader(tc.customerCode)
						return
					}
					externalID, customerID := orgID.String(), "cust_live"
					if tc.foreignCustomer {
						externalID = "another-organization"
					}
					if tc.emptyCustomer {
						customerID = ""
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"id": customerID, "external_id": externalID}))
				case "/customers/external/" + orgID.String() + "/payment-methods":
					if tc.cardCode != 0 {
						w.WriteHeader(tc.cardCode)
						return
					}
					items := []map[string]any{}
					if !tc.noCard && !removedCard.Load() {
						expiry := time.Now().AddDate(1, 0, 0)
						if tc.expired {
							expiry = time.Now().AddDate(-1, 0, 0)
						}
						items = append(items, map[string]any{"id": "card", "type": "card", "method_metadata": map[string]int{"exp_month": int(expiry.Month()), "exp_year": expiry.Year()}})
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items}))
				case "/subscriptions/":
					assert.Equal(t, orgID.String(), req.URL.Query().Get("external_customer_id"))
					if tc.subscriptionCode != 0 {
						w.WriteHeader(tc.subscriptionCode)
						return
					}
					_, _ = w.Write([]byte(`{"items":[{"id":"sub_live","status":"paused"}]}`))
				default:
					t.Errorf("unexpected Polar request: %s", req.URL)
					http.NotFound(w, req)
				}
			}))
			t.Cleanup(upstream.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "test")
			if tc.notConfigured {
				t.Setenv("POLAR_ACCESS_TOKEN", "")
			}
			t.Setenv("POLAR_API_BASE_URL", upstream.URL)
			path := "/admin/api/organizations/" + orgID.String()
			if tc.precheck {
				checked := execRequest(server, requestParams{method: "GET", path: path + "/billing-plan?check_polar=true", authCookie: token})
				require.Equal(t, 200, checked.Code)
				var diagnostics organizationBillingPlanResponse
				require.NoError(t, json.Unmarshal(checked.Body.Bytes(), &diagnostics))
				require.Equal(t, "usable", diagnostics.Polar.CardStatus)
				removedCard.Store(true)
			}
			response := execRequest(server, requestParams{method: "POST", path: path + "/billing-customer/sync", authCookie: token})
			require.Equal(t, tc.status, response.Code, response.Body.String())
			var result struct {
				BillingPlan organizationBillingPlanResponse `json:"billing_plan"`
				Error       string                          `json:"error"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			currentSettings, err := billingmodels.FindOrganizationLLMSettings(db, orgID)
			require.NoError(t, err)
			if tc.status == 200 {
				require.NotNil(t, currentSettings.PolarCustomerID)
				assert.Equal(t, "cust_live", *currentSettings.PolarCustomerID)
				if tc.noSettings {
					assert.Nil(t, currentSettings.MarkupBPS)
				} else {
					assert.Equal(t, &markup, currentSettings.MarkupBPS)
				}
				assert.Equal(t, "cust_live", result.BillingPlan.PolarCustomerID)
				assert.Equal(t, "usable", result.BillingPlan.Polar.CardStatus)
				assert.False(t, result.BillingPlan.Polar.CustomerIDMismatch)
				assert.Empty(t, result.Error)
				if tc.subscriptionCode != 0 {
					assert.Equal(t, "failed", result.BillingPlan.PolarSubscriptionVerification)
				} else {
					assert.Equal(t, "active", result.BillingPlan.PolarSubscriptionVerification)
				}
			} else {
				assert.Equal(t, previousSettings, currentSettings)
				assert.Equal(t, tc.storedID, result.BillingPlan.PolarCustomerID)
				assert.NotEmpty(t, result.Error)
			}
			currentPlan, err := billingmodels.FindOrganizationBillingPlan(db, orgID)
			require.NoError(t, err)
			assert.Equal(t, previousPlan, currentPlan)
			assert.Equal(t, !tc.missingPlan, result.BillingPlan.RecordExists)
			if tc.notConfigured {
				assert.Zero(t, requests.Load())
			}
			if tc.missingPlan && tc.status == http.StatusOK {
				// The next organization billing read must discover the existing card
				// through the repaired mapping, without storing a local card override.
				billing, err := billingorganizations.DescribeOrganizationBilling(t.Context(), orgID.String(), &pb.DescribeOrganizationBillingRequest{})
				require.NoError(t, err)
				assert.True(t, billing.HasBillingCustomer)
				assert.True(t, billing.HasPaymentMethod)
			}
		})
	}
}
