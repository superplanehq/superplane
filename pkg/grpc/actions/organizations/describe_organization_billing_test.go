package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__SyncOrganizationBilling(t *testing.T) {
	r := support.Setup(t)

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
			assert.Equal(t, "/subscriptions/", req.URL.Path)
			assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{
						"id":                   "sub_grpc_sync",
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
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
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
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/subscriptions/sub_grpc_cancel", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_cancel",
				"status":               "active",
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
	db := database.Conn()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
		ID:                "sub_grpc_resume",
		Status:            models.PolarSubscriptionStatusActive,
		PeriodStart:       &periodStart,
		PeriodEnd:         &periodEnd,
		CancelAtPeriodEnd: true,
	})
	require.NoError(t, err)

	t.Run("keeps Business", func(t *testing.T) {
		server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/subscriptions/sub_grpc_resume", req.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":                   "sub_grpc_resume",
				"status":               "active",
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
