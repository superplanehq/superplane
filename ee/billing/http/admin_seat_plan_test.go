package billinghttp

import (
	"encoding/json"
	sentrytest "github.com/superplanehq/superplane/test/support/sentry"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
)

const monthlySeatProduct = `{"id":"prod_team","is_archived":false,"is_recurring":true,"recurring_interval":"month","recurring_interval_count":1,"prices":[{"amount_type":"seat_based","is_archived":false,"price_currency":"usd","seat_tiers":{"seat_tier_type":"volume","minimum_seats":1,"maximum_seats":1000,"tiers":[{"min_seats":1,"max_seats":1000,"price_per_seat":2900}]}}]}`

func TestAdminSeatPlanProductValidation(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	db := database.Conn()
	reports := sentrytest.BindHub(t)
	for _, tc := range []struct {
		name       string
		product    string
		status     int
		wantStatus int
		message    string
	}{
		{"valid", monthlySeatProduct, 200, 200, ""},
		{"unlimited Polar tier", strings.ReplaceAll(monthlySeatProduct, `1000`, `null`), 200, 200, ""},
		{"archived historical price", strings.Replace(monthlySeatProduct, `"prices":[`, `"prices":[{"amount_type":"fixed","is_archived":true},`, 1), 200, 200, ""},
		{"one-time", strings.Replace(monthlySeatProduct, `"is_recurring":true`, `"is_recurring":false`, 1), 200, 400, "monthly subscription"},
		{"yearly", strings.Replace(monthlySeatProduct, `"month"`, `"year"`, 1), 200, 400, "monthly subscription"},
		{"every two months", strings.Replace(monthlySeatProduct, `"recurring_interval_count":1`, `"recurring_interval_count":2`, 1), 200, 400, "monthly subscription"},
		{"missing interval count", strings.Replace(monthlySeatProduct, `,"recurring_interval_count":1`, ``, 1), 200, 400, "monthly subscription"},
		{"archived product", strings.Replace(monthlySeatProduct, `"is_archived":false`, `"is_archived":true`, 1), 200, 400, "archived"},
		{"archived price", strings.Replace(monthlySeatProduct, `"amount_type":"seat_based","is_archived":false`, `"amount_type":"seat_based","is_archived":true`, 1), 200, 400, "one active USD seat-based price"},
		{"no prices", strings.Replace(monthlySeatProduct, `"prices":[`, `"prices":[],"ignored":[`, 1), 200, 400, "one active USD seat-based price"},
		{"flat price", strings.Replace(monthlySeatProduct, `"seat_based"`, `"fixed"`, 1), 200, 400, "one active USD seat-based price"},
		{"wrong currency", strings.Replace(monthlySeatProduct, `"usd"`, `"eur"`, 1), 200, 400, "one active USD seat-based price"},
		{"additional charge", strings.Replace(monthlySeatProduct, `"prices":[`, `"prices":[{"amount_type":"fixed","is_archived":false,"price_amount":500},`, 1), 200, 400, "one active USD seat-based price"},
		{"missing seat tiers", strings.Replace(monthlySeatProduct, `"seat_tiers":`, `"ignored":`, 1), 200, 400, "one fixed-rate seat tier"},
		{"variable tiers", strings.Replace(monthlySeatProduct, `"tiers":[`, `"tiers":[{"min_seats":1,"max_seats":10,"price_per_seat":2900},`, 1), 200, 400, "one fixed-rate seat tier"},
		{"minimum above one", strings.Replace(monthlySeatProduct, `"min_seats":1`, `"min_seats":2`, 1), 200, 400, "cover seats 1 through"},
		{"maximum too low", strings.ReplaceAll(monthlySeatProduct, `1000`, `49`), 200, 400, "cover seats 1 through"},
		{"different seat price", strings.Replace(monthlySeatProduct, `2900`, `3000`, 1), 200, 400, "catalog price"},
		{"flat amount is not a seat price", strings.Replace(monthlySeatProduct, `"price_per_seat":2900`, `"price_amount":2900`, 1), 200, 400, "catalog price"},
		{"missing product", `private provider detail`, 404, 400, "not found"},
		{"provider failure", `private provider detail`, 503, 502, "try again"},
		{"unauthorized", `private provider detail`, 401, 502, "try again"},
		{"rate limited", `private provider detail`, 429, 502, "try again"},
		{"invalid JSON", `private provider detail`, 200, 502, "try again"},
		{"empty response", ``, 200, 502, "try again"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			productID := "prod_" + uuid.NewString()
			var calls atomic.Int32
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/products/"+productID, r.URL.Path)
				assert.Equal(t, "Bearer oat_test", r.Header.Get("Authorization"))
				w.Header().Set("Polar-Version", r.Header.Get("Polar-Version"))
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(strings.ReplaceAll(tc.product, "prod_team", productID)))
			}))
			t.Cleanup(provider.Close)
			t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
			t.Setenv("POLAR_API_BASE_URL", provider.URL)
			for _, operation := range []string{"create", "update"} {
				t.Run(operation, func(t *testing.T) {
					plan := billingmodels.BillingPlan{Key: "team_" + strings.ReplaceAll(uuid.NewString(), "-", ""), Name: "Team", Category: "paid", Visibility: "public", PriceCents: 2900, PerSeat: true, SeatLimit: new(50), PolarProductID: &productID, CreditFrequency: "none", CreditExpiration: "never"}
					t.Cleanup(func() { require.NoError(t, db.Where("key = ?", plan.Key).Delete(&billingmodels.BillingPlan{}).Error) })
					method, path := "POST", "billing-plans"
					var original *billingmodels.BillingPlan
					if operation == "update" {
						plan.SeatLimit = new(1)
						require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
						var err error
						original, err = billingmodels.FindBillingPlan(db, plan.Key)
						require.NoError(t, err)
						plan.SeatLimit = new(50)
						plan.Name = "Team updated"
						method, path = "PUT", path+"/"+plan.ID.String()
					}
					body, err := json.Marshal(plan)
					require.NoError(t, err)
					before := calls.Load()
					reportsBefore := len(reports.Events())
					response := request(method, path, string(body))
					require.Equal(t, before+1, calls.Load(), "every save must fetch the product")
					require.NotContains(t, response.Body.String(), "private provider detail")
					reported := reports.Events()[reportsBefore:]
					if tc.wantStatus == http.StatusOK {
						require.Equal(t, http.StatusOK, response.Code, response.Body.String())
						require.Empty(t, reported)
						stored, err := billingmodels.FindBillingPlan(db, plan.Key)
						require.NoError(t, err)
						require.Equal(t, 50, *stored.SeatLimit)
						require.Equal(t, plan.Name, stored.Name)
						return
					}
					require.Contains(t, requireBillingPlanJSONError(t, response, tc.wantStatus), tc.message)
					if tc.wantStatus == http.StatusBadGateway {
						require.Len(t, reported, 1, "provider failures keep their cause in the error report")
						require.NotEmpty(t, sentrytest.ExceptionValues(reported[0]))
					} else {
						require.Empty(t, reported)
					}
					if operation == "create" {
						var count int64
						require.NoError(t, db.Model(&billingmodels.BillingPlan{}).Where("key = ?", plan.Key).Count(&count).Error)
						require.Zero(t, count)
						return
					}
					stored, err := billingmodels.FindBillingPlan(db, plan.Key)
					require.NoError(t, err)
					require.Equal(t, original, stored, "failed validation must not change any stored fields")
				})
			}
		})
	}
}

func TestAdminUnlinkedSeatPlansDoNotRequirePolar(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	t.Setenv("POLAR_ACCESS_TOKEN", "")
	key := "unlinked_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	plan := createAdminTestPlan(t, request, `{"key":"`+key+`","name":"Team","category":"paid","visibility":"private","per_seat":true,"seat_limit":50,"credit_frequency":"none","credit_expiration":"never"}`)
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("id = ?", plan.ID).Delete(&billingmodels.BillingPlan{}).Error)
	})
	plan.PolarProductID = new("prod_team")
	body, err := json.Marshal(plan)
	require.NoError(t, err)
	response := request("PUT", "billing-plans/"+plan.ID.String(), string(body))
	require.Contains(t, requireBillingPlanJSONError(t, response, http.StatusBadRequest), "Configure Polar")
	stored, err := billingmodels.FindBillingPlan(database.Conn(), plan.Key)
	require.NoError(t, err)
	require.Nil(t, stored.PolarProductID)
}
