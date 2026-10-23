package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DescribeOrganizationBillingCatalogPlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
	catalog := models.BillingPlan{
		Key: "billing_response_custom", Name: "Team", Category: "paid", Visibility: "private",
		PriceCents: 2900, PerSeat: true, SeatLimit: new(50), CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, models.SaveBillingPlan(database.Conn(), &catalog))
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBillingPlan{}).Error)
		require.NoError(t, models.DeleteBillingPlan(database.Conn(), catalog.ID))
	})
	_, err := models.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, catalog.Key, nil)
	require.NoError(t, err)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, catalog.Key, resp.Plan)
	assert.Equal(t, "Team", resp.PlanName)
	assert.Equal(t, "paid", resp.PlanCategory)
	assert.Equal(t, int64(2900), resp.PlanPriceCents)
	assert.True(t, resp.PlanPerSeat)

	db := database.Conn()
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 10).Error)
	for range 8 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	expected := map[string]any{
		"purchasedSeats": float64(10), "occupiedSeats": float64(9), "availableSeats": float64(1),
		"planMaximumSeats": float64(50), "purchasableSeats": float64(40), "expectedRenewalSeats": float64(9),
		"currentMonthlyTotalCents": "29000", "expectedRenewalCents": "26100",
	}
	check := func() {
		t.Helper()
		resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
		require.NoError(t, err)
		assertBillingSeatJSON(t, resp, expected)
	}
	check()

	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, new(80)))
	expected["planMaximumSeats"], expected["purchasableSeats"] = float64(80), float64(70)
	check()

	require.NoError(t, db.Delete(&models.User{}, "id = ?", r.User).Error)
	expected["occupiedSeats"], expected["availableSeats"], expected["expectedRenewalSeats"] = float64(8), float64(2), float64(8)
	expected["expectedRenewalCents"] = "23200"
	check()

	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", nil).Error)
	expected["purchasedSeats"], expected["availableSeats"], expected["purchasableSeats"] = float64(0), float64(0), float64(80)
	expected["currentMonthlyTotalCents"] = "0"
	check()

	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, nil))
	delete(expected, "planMaximumSeats")
	delete(expected, "purchasableSeats")
	check()

	require.NoError(t, db.Delete(&models.User{}, "organization_id = ?", r.Organization.ID).Error)
	expected["occupiedSeats"], expected["expectedRenewalSeats"] = float64(0), float64(1)
	expected["expectedRenewalCents"] = "2900"
	check()
}

// Use the gateway marshaler to distinguish optional zeros from omitted fields.
func assertBillingSeatJSON(t *testing.T, resp *pb.DescribeOrganizationBillingResponse, expected map[string]any) {
	t.Helper()
	marshaler := runtime.JSONPb{MarshalOptions: protojson.MarshalOptions{EmitUnpopulated: true}}
	body, err := marshaler.Marshal(resp)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(body, &fields))
	for _, key := range []string{"purchasedSeats", "occupiedSeats", "availableSeats", "planMaximumSeats", "purchasableSeats", "expectedRenewalSeats", "currentMonthlyTotalCents", "expectedRenewalCents"} {
		value, present := expected[key]
		if !present {
			assert.NotContains(t, fields, key)
			continue
		}
		require.Contains(t, fields, key)
		assert.Equal(t, value, fields[key], key)
	}
}

func Test__DescribeOrganizationBillingOccupiedOnNonPerSeatPlans(t *testing.T) {
	for _, key := range []string{"business_legacy", "trial", "none"} {
		t.Run(key, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
			db := database.Conn()
			catalog, err := models.FindBillingPlan(db, key)
			require.NoError(t, err)
			require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
				"plan_id": catalog.ID, "plan": catalog.ResponsePlan(), "plan_source": "admin",
				"trial_ends_at": time.Now().Add(time.Hour), "purchased_seats": 99,
			}).Error)
			support.CreateUser(t, r, r.Organization.ID)
			resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			assert.False(t, resp.PlanPerSeat)
			assertBillingSeatJSON(t, resp, map[string]any{"occupiedSeats": float64(2)})
		})
	}
}

func Test__DescribeOrganizationBillingDisabledWhenFlagOff(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	t.Setenv(models.BillingEnabledEnv, "no")
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.False(t, resp.BillingEnabled)
	assert.False(t, resp.SubscriptionCheckoutEnabled)
	assertBillingSeatJSON(t, resp, nil)

	_, err = SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = CreateBusinessCheckout(context.Background(), r.Organization.ID.String(), &pb.CreateBusinessCheckoutRequest{}, r.Account.ID.String(), "http://localhost")
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = CancelOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))

	_, err = ResumeOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}

func Test__SyncOrganizationBilling(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := SyncOrganizationBilling(context.Background(), "not-a-uuid", &pb.SyncOrganizationBillingRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("returns local plan when billing is not configured", func(t *testing.T) {
		t.Setenv("POLAR_ACCESS_TOKEN", "")
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "")

		resp, err := SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.BillingPlanTrial, resp.Plan)
		assert.False(t, resp.CreditPurchaseAllowed)
	})

	t.Run("applies an active Polar subscription", func(t *testing.T) {
		periodStart := time.Now().UTC().Truncate(time.Second)
		periodEnd := periodStart.AddDate(0, 1, 0)
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			assert.Equal(t, "/subscriptions/", req.URL.Path)
			assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":                   "sub_grpc_sync",
						"status":               "active",
						"product_id":           "prod_business",
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
		})
		usePolarTestServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := SyncOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.Equal(t, "active", resp.PolarSubscriptionStatus)
	})

	t.Run("returns local plan when Polar fails", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("billing-sync"), "")
		require.NoError(t, err)

		server := polarAPIServer(t, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "polar down", http.StatusInternalServerError)
		})
		usePolarTestServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := SyncOrganizationBilling(context.Background(), other.ID.String(), &pb.SyncOrganizationBillingRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.BillingPlanTrial, resp.Plan)
		assert.False(t, resp.CreditPurchaseAllowed)
	})
}

func Test__DescribeOrganizationBillingLapsesExpiredTrial(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	ended := time.Now().Add(-time.Hour)
	require.NoError(t, database.Conn().Model(&models.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{
			"plan":          models.BillingPlanTrial,
			"trial_ends_at": ended,
		}).Error)

	resp, err := DescribeOrganizationBilling(context.Background(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, resp.Plan)
	assert.False(t, resp.CreditPurchaseAllowed)

	stored, err := models.FindOrganizationBillingPlan(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.BillingPlanNone, stored.Plan)
}

func Test__CancelOrganizationSubscription(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{ProductID: "prod_business",
		ID:          "sub_grpc_cancel",
		Status:      models.PolarSubscriptionStatusActive,
		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
	})
	require.NoError(t, err)

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := CancelOrganizationSubscription(context.Background(), "bad", &pb.CancelOrganizationSubscriptionRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("rejects trial", func(t *testing.T) {
		other, err := models.CreateOrganization(support.RandomName("billing-cancel"), "")
		require.NoError(t, err)
		t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		_, err = CancelOrganizationSubscription(context.Background(), other.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	})

	t.Run("cancels at period end", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/subscriptions/sub_grpc_cancel", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_cancel",
				"status":               "active",
				"product_id":           "prod_business",
				"cancel_at_period_end": true,
				"current_period_start": periodStart.Format(time.RFC3339),
				"current_period_end":   periodEnd.Format(time.RFC3339),
				"external_customer_id": r.Organization.ID.String(),
				"customer": map[string]any{
					"id":          "cust_polar_1",
					"external_id": r.Organization.ID.String(),
				},
			}))
		})
		usePolarTestServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := CancelOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.CancelOrganizationSubscriptionRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.True(t, resp.CancelAtPeriodEnd)
	})
}

func Test__ResumeOrganizationSubscription(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{ProductID: "prod_business",
		ID:                "sub_grpc_resume",
		Status:            models.PolarSubscriptionStatusActive,
		PeriodStart:       &periodStart,
		PeriodEnd:         &periodEnd,
		CancelAtPeriodEnd: true,
	})
	require.NoError(t, err)

	t.Run("keeps Business", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/payment-methods") {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}}))
				return
			}
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/subscriptions/sub_grpc_resume", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_resume",
				"status":               "active",
				"product_id":           "prod_business",
				"cancel_at_period_end": false,
				"current_period_start": periodStart.Format(time.RFC3339),
				"current_period_end":   periodEnd.Format(time.RFC3339),
				"external_customer_id": r.Organization.ID.String(),
				"customer": map[string]any{
					"id":          "cust_polar_1",
					"external_id": r.Organization.ID.String(),
				},
			}))
		})
		usePolarTestServer(t, server)
		t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")

		resp, err := ResumeOrganizationSubscription(context.Background(), r.Organization.ID.String(), &pb.ResumeOrganizationSubscriptionRequest{})
		require.NoError(t, err)
		assert.Equal(t, models.BillingPlanBusiness, resp.Plan)
		assert.True(t, resp.CreditPurchaseAllowed)
		assert.False(t, resp.CancelAtPeriodEnd)
	})
}

func TestBusinessCheckoutSuccessURLIncludesCheckoutID(t *testing.T) {
	id := uuid.New()
	assert.Equal(t, "https://app.superplane.com/"+id.String()+"/organization/billing?subscribed=1&checkout_id={CHECKOUT_ID}", businessCheckoutSuccessURL("https://app.superplane.com/", id))
}

func TestLegacyBusinessCheckoutRemainsAvailable(t *testing.T) {
	for _, tc := range []struct {
		name, catalogProduct, envProduct, wantProduct string
	}{
		{"environment fallback", "", "prod_business", "prod_business"},
		{"catalog without environment", "prod_catalog", "", "prod_catalog"},
		{"catalog overrides environment", "prod_catalog", "prod_environment", "prod_catalog"},
		{"unconfigured", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			catalog, err := models.FindBillingPlan(db, models.BillingPlanBusiness)
			require.NoError(t, err)
			originalProduct := catalog.PolarProductID
			t.Cleanup(func() {
				require.NoError(t, database.Conn().Model(catalog).Update("polar_product_id", originalProduct).Error)
			})
			var catalogProduct *string
			if tc.catalogProduct != "" {
				catalogProduct = &tc.catalogProduct
			}
			require.NoError(t, db.Model(catalog).Update("polar_product_id", catalogProduct).Error)
			t.Setenv(models.BillingEnabledEnv, "yes")
			t.Setenv("POLAR_BUSINESS_PRODUCT_ID", tc.envProduct)
			server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/customers/external/" + r.Organization.ID.String():
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"id": "cust_legacy", "external_id": r.Organization.ID.String()}))
				case "/checkouts/":
					var body map[string]any
					require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
					require.Equal(t, []any{tc.wantProduct}, body["products"])
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"url": "https://polar.sh/checkout/legacy"}))
				default:
					t.Errorf("unexpected Polar request %s", req.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
				}
			})
			usePolarTestServer(t, server)
			response, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
			require.NoError(t, err)
			require.True(t, response.BillingEnabled)
			require.Equal(t, tc.wantProduct != "", response.SubscriptionCheckoutEnabled)
			checkout, err := CreateBusinessCheckout(t.Context(), r.Organization.ID.String(), &pb.CreateBusinessCheckoutRequest{}, r.Account.ID.String(), "http://localhost")
			if tc.wantProduct == "" {
				require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, "https://polar.sh/checkout/legacy", checkout.CheckoutUrl)
		})
	}
}

func TestDescribeBillingListsPublicPlansWithoutPolarProducts(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	t.Setenv(models.BillingEnabledEnv, "yes")
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	plans := []models.BillingPlan{
		{Name: "Flat price", Visibility: "public", PriceCents: 2000},
		{Name: "Seat price", Visibility: "public", PriceCents: 3000, PerSeat: true, SeatLimit: new(50)},
		{Name: "Private", Visibility: "private", PriceCents: 1000},
	}
	for i := range plans {
		plans[i].Key = "catalog_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		plans[i].Category = "paid"
		plans[i].CreditFrequency = "none"
		plans[i].CreditExpiration = "never"
		require.NoError(t, models.SaveBillingPlan(db, &plans[i]))
	}
	response, err := DescribeOrganizationBilling(t.Context(), r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	for _, plan := range plans {
		var option *pb.BillingPlanOption
		for _, available := range response.AvailablePlans {
			if available.Key == plan.Key {
				option = available
				break
			}
		}
		if plan.Visibility == "private" {
			require.Nil(t, option)
			continue
		}
		require.NotNil(t, option, "public plans must remain visible before Polar setup")
		require.Equal(t, plan.PriceCents, option.PriceCents)
		require.Equal(t, plan.PerSeat, option.PerSeat)
		require.True(t, option.SubscriptionRequired, "priced plans must never use card-only activation")
		_, err := SelectBillingPlan(t.Context(), r.Organization.ID.String(), &pb.SelectBillingPlanRequest{PlanKey: plan.Key}, r.Account.ID.String(), "http://localhost")
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		current, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
		require.NoError(t, err)
		require.NotEqual(t, plan.ID, current.PlanID)
	}
}
