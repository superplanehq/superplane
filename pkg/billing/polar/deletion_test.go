package polar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const deletionSubscriptionID = "sub_deletion"

// storeElapsedScheduledCancellation leaves the local row as a delayed event
// would: canceling at a period end that passed, with a non-terminal status.
func storeElapsedScheduledCancellation(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	end := time.Now().Add(-time.Hour)
	start := end.AddDate(0, -1, 0)
	_, err := models.ApplyPolarSubscription(database.Conn(), orgID, models.PolarSubscriptionApply{
		ID: deletionSubscriptionID, ProductID: "prod_business", Status: models.PolarSubscriptionStatusActive,
		PeriodStart: &start, PeriodEnd: &end, CancelAtPeriodEnd: true,
	}, nil)
	require.NoError(t, err)
}

func providerSubscription(orgID uuid.UUID, id, status string, cancelAtPeriodEnd bool) map[string]any {
	return map[string]any{
		"id": id, "status": status, "cancel_at_period_end": cancelAtPeriodEnd,
		"current_period_end":   time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"external_customer_id": orgID.String(),
	}
}

func TestVerifyOrganizationDeletionAsksPolarAboutStaleCancellation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		providers func(orgID uuid.UUID) []map[string]any
		err       error
	}{
		{
			name: "provider resumed the subscription",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "active", false)}
			},
			err: models.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider reports past due",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "past_due", false)}
			},
			err: models.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider reports paused",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "paused", false)}
			},
			err: models.ErrOrganizationSubscriptionLive,
		},
		{
			name: "another subscription is live",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{
					providerSubscription(orgID, deletionSubscriptionID, "canceled", false),
					providerSubscription(orgID, "sub_other", "active", false),
				}
			},
			err: models.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider does not return the stored subscription",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, "sub_other", "canceled", false)}
			},
			err: models.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "provider returns another customer",
			providers: func(uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(uuid.New(), deletionSubscriptionID, "canceled", false)}
			},
			err: models.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "provider returns an unknown status",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "suspended", false)}
			},
			err: models.ErrOrganizationSubscriptionUnverified,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			storeElapsedScheduledCancellation(t, org.ID)
			usePolarSubscriptionServer(t, org.ID.String(), tc.providers(org.ID))

			clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)

			require.ErrorIs(t, err, tc.err)
			assert.Nil(t, clearance)
		})
	}
}

func TestVerifyOrganizationDeletionClearsProviderTermination(t *testing.T) {
	for _, status := range []string{"canceled", "revoked"} {
		t.Run(status, func(t *testing.T) {
			org := setupBillingOrganization(t)
			storeElapsedScheduledCancellation(t, org.ID)
			usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
				providerSubscription(org.ID, deletionSubscriptionID, status, true),
			})

			clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)
			require.NoError(t, err)
			require.NotNil(t, clearance)

			require.NoError(t, org.SoftDelete(database.DB(t.Context()), clearance))
			_, err = models.FindOrganizationByID(org.ID.String())
			require.Error(t, err)
		})
	}
}

func TestVerifyOrganizationDeletionFailsClosedWhenPolarFails(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		org := setupBillingOrganization(t)
		storeElapsedScheduledCancellation(t, org.ID)
		usePolarHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))

		_, err := VerifyOrganizationDeletion(context.Background(), org.ID)

		require.ErrorIs(t, err, models.ErrOrganizationSubscriptionUnverified)
	})

	t.Run("timeout", func(t *testing.T) {
		org := setupBillingOrganization(t)
		storeElapsedScheduledCancellation(t, org.ID)
		usePolarHandler(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		_, err := VerifyOrganizationDeletion(ctx, org.ID)

		require.ErrorIs(t, err, models.ErrOrganizationSubscriptionUnverified)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("polar is not configured", func(t *testing.T) {
		org := setupBillingOrganization(t)
		storeElapsedScheduledCancellation(t, org.ID)
		t.Setenv("POLAR_ACCESS_TOKEN", "")

		_, err := VerifyOrganizationDeletion(context.Background(), org.ID)

		require.ErrorIs(t, err, models.ErrOrganizationSubscriptionUnverified)
	})
}

func TestVerifyOrganizationDeletionSkipsPolarForPlanThatCannotBill(t *testing.T) {
	org := setupBillingOrganization(t)
	usePolarHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Polar request %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)

	require.NoError(t, err)
	require.NotNil(t, clearance)
	require.NoError(t, org.SoftDelete(database.DB(t.Context()), clearance))
}

func usePolarHandler(t *testing.T, handler http.Handler) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
}
