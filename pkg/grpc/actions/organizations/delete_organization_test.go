package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func Test__DeleteOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, r.AuthService, uuid.New().String())
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "organization not found", msg)
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := DeleteOrganization(context.Background(), r.AuthService, r.Organization.ID.String())
		require.Error(t, err)
		assert.ErrorContains(t, err, "user not authenticated")
	})

	t.Run("organization is deleted", func(t *testing.T) {
		response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		assert.Error(t, err)
	})
}

func Test__DeleteOrganization_TransactionRollback(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("auth service failure rolls back organization soft-deletion", func(t *testing.T) {
		//
		// Verify organization exists and is not soft-deleted
		//
		foundOrg, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)

		//
		// Use an authentication service that fails
		//
		mockAuth := &mockAuthService{
			Authorization: r.AuthService,
			Error:         errors.New("ooops"),
		}

		//
		// Try to delete organization
		// It should fail due to destroy organization error.
		//
		_, err = DeleteOrganization(ctx, mockAuth, r.Organization.ID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ooops")

		//
		// Verify organization is NOT soft-deleted after transaction rollback
		//
		foundOrg, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)
	})
}

func Test__DeleteOrganizationCancelsActiveBusinessPlan(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()
	activateBusinessPlan(t, db, r.Organization.ID, "sub_delete")

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Equal(t, "/subscriptions/sub_delete", req.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		assert.Equal(t, true, body["cancel_at_period_end"])
		periodEnd := time.Now().UTC().AddDate(0, 1, 0)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_delete",
			"status":               "active",
			"cancel_at_period_end": true,
			"current_period_end":   periodEnd.Format(time.RFC3339),
			"external_customer_id": r.Organization.ID.String(),
			"customer": map[string]any{
				"id":          "cust_polar_1",
				"external_id": r.Organization.ID.String(),
			},
		}))
	}))
	t.Cleanup(server.Close)
	enablePolar(t, server.URL)

	response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, 1, calls)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	assert.Error(t, err)

	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.True(t, plan.CancelAtPeriodEnd)
}

func Test__DeleteOrganizationKeepsOrganizationWhenPlanCancelFails(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()
	activateBusinessPlan(t, db, r.Organization.ID, "sub_down")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "polar down", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	enablePolar(t, server.URL)

	_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.Error(t, err)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, code)
	assert.Equal(t, "failed to cancel the Business plan. The organization was not deleted.", msg)

	found, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.False(t, found.DeletedAt.Valid)
}

func activateBusinessPlan(t *testing.T, db *gorm.DB, orgID uuid.UUID, subscriptionID string) {
	t.Helper()
	periodStart := time.Now().UTC().Truncate(time.Second)
	periodEnd := periodStart.AddDate(0, 1, 0)
	_, _, err := models.ApplyPolarSubscription(db, orgID, models.PolarSubscriptionApply{
		ID:          subscriptionID,
		Status:      models.PolarSubscriptionStatusActive,
		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
	})
	require.NoError(t, err)
}

func enablePolar(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	t.Setenv("POLAR_API_BASE_URL", baseURL)
}
