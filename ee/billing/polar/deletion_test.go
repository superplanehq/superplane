package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

const deletionSubscriptionID = "sub_deletion"

// storeElapsedScheduledCancellation leaves the local row as a delayed event
// would: canceling at a period end that passed, with a non-terminal status.
func storeElapsedScheduledCancellation(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	end := time.Now().Add(-time.Hour)
	start := end.AddDate(0, -1, 0)
	_, err := billingmodels.ApplyPolarSubscription(database.Conn(), orgID, billingmodels.PolarSubscriptionApply{
		ID: deletionSubscriptionID, ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
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
			err: billingmodels.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider reports past due",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "past_due", false)}
			},
			err: billingmodels.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider reports paused",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "paused", false)}
			},
			err: billingmodels.ErrOrganizationSubscriptionLive,
		},
		{
			name: "another subscription is live",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{
					providerSubscription(orgID, deletionSubscriptionID, "canceled", false),
					providerSubscription(orgID, "sub_other", "active", false),
				}
			},
			err: billingmodels.ErrOrganizationSubscriptionLive,
		},
		{
			name: "another subscription is unpaid",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{
					providerSubscription(orgID, deletionSubscriptionID, "canceled", false),
					providerSubscription(orgID, "sub_other", "unpaid", false),
				}
			},
			err: billingmodels.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "another subscription is incomplete",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{
					providerSubscription(orgID, deletionSubscriptionID, "canceled", false),
					providerSubscription(orgID, "sub_other", "incomplete", false),
				}
			},
			err: billingmodels.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "another subscription has an ended timestamp but a non-terminal status",
			providers: func(orgID uuid.UUID) []map[string]any {
				other := providerSubscription(orgID, "sub_other", "active", false)
				other["ended_at"] = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
				return []map[string]any{
					providerSubscription(orgID, deletionSubscriptionID, "canceled", false),
					other,
				}
			},
			err: billingmodels.ErrOrganizationSubscriptionLive,
		},
		{
			name: "provider does not return the stored subscription",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, "sub_other", "canceled", false)}
			},
			err: billingmodels.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "provider returns another customer",
			providers: func(uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(uuid.New(), deletionSubscriptionID, "canceled", false)}
			},
			err: billingmodels.ErrOrganizationSubscriptionUnverified,
		},
		{
			name: "provider returns an unknown status",
			providers: func(orgID uuid.UUID) []map[string]any {
				return []map[string]any{providerSubscription(orgID, deletionSubscriptionID, "suspended", false)}
			},
			err: billingmodels.ErrOrganizationSubscriptionUnverified,
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
	for _, status := range []string{"canceled", "revoked", "incomplete_expired"} {
		t.Run(status, func(t *testing.T) {
			org := setupBillingOrganization(t)
			storeElapsedScheduledCancellation(t, org.ID)
			usePolarSubscriptionServer(t, org.ID.String(), []map[string]any{
				providerSubscription(org.ID, deletionSubscriptionID, status, true),
				providerSubscription(org.ID, "sub_other", "canceled", false),
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

		require.ErrorIs(t, err, billingmodels.ErrOrganizationSubscriptionUnverified)
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

		require.ErrorIs(t, err, billingmodels.ErrOrganizationSubscriptionUnverified)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("polar is not configured", func(t *testing.T) {
		org := setupBillingOrganization(t)
		storeElapsedScheduledCancellation(t, org.ID)
		t.Setenv("POLAR_ACCESS_TOKEN", "")

		_, err := VerifyOrganizationDeletion(context.Background(), org.ID)

		require.ErrorIs(t, err, billingmodels.ErrOrganizationSubscriptionUnverified)
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

// useCheckoutServer answers checkout reads with the given status.
func useCheckoutServer(t *testing.T, checkoutID, status string, statusCode int) {
	t.Helper()
	usePolarHandler(t, echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			return
		}
		assert.Equal(t, "/checkouts/"+checkoutID, r.URL.Path)
		payload := checkoutPayload(checkoutID, status)
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
}

func TestVerifyOrganizationDeletionRefusesACheckoutThatCanStillActivate(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     string
		statusCode int
		err        error
	}{
		{name: "open checkout", status: "open", statusCode: http.StatusOK, err: billingmodels.ErrOrganizationCheckoutPending},
		{name: "confirmed checkout", status: "confirmed", statusCode: http.StatusOK, err: billingmodels.ErrOrganizationCheckoutPending},
		{name: "Polar cannot answer", statusCode: http.StatusBadGateway, err: billingmodels.ErrOrganizationSubscriptionUnverified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			intent := openTeamCheckout(t, org, "chk_delete")
			useCheckoutServer(t, "chk_delete", tc.status, tc.statusCode)

			clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)

			require.ErrorIs(t, err, tc.err)
			assert.Nil(t, clearance)
			stored := unresolvedCheckout(t, org)
			require.NotNil(t, stored)
			assert.Equal(t, intent.ID, stored.ID)
		})
	}
}

func TestVerifyOrganizationDeletionReleasesAVerifiedExpiredCheckout(t *testing.T) {
	org := setupBillingOrganization(t)
	openTeamCheckout(t, org, "chk_delete")
	useCheckoutServer(t, "chk_delete", "expired", http.StatusOK)

	clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)
	require.NoError(t, err)

	require.NoError(t, org.SoftDelete(database.DB(t.Context()), clearance))
	_, err = models.FindOrganizationByID(org.ID.String())
	require.Error(t, err)
}

func TestSoftDeleteRefusesACheckoutClaimedAfterTheVerification(t *testing.T) {
	org := setupBillingOrganization(t)
	clearance, err := VerifyOrganizationDeletion(context.Background(), org.ID)
	require.NoError(t, err)
	// A checkout starts between the verification and the deletion.
	openTeamCheckout(t, org, "chk_late")

	err = org.SoftDelete(database.DB(t.Context()), clearance)

	require.ErrorIs(t, err, billingmodels.ErrOrganizationCheckoutPending)
	_, err = models.FindOrganizationByID(org.ID.String())
	require.NoError(t, err, "the organization stays while its checkout can activate")
}

func TestVerifyOrganizationDeletionChecksPreexistingCheckouts(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       string
		fails        bool
		unconfigured bool
		err          error
	}{
		{name: "open legacy checkout", status: "open", err: billingmodels.ErrOrganizationCheckoutPending},
		{name: "confirmed legacy checkout", status: "confirmed", err: billingmodels.ErrOrganizationCheckoutPending},
		{name: "checkout expires during verification", status: "expired"},
		{name: "no legacy checkout"},
		{name: "Polar cannot list legacy checkouts", fails: true, err: billingmodels.ErrOrganizationSubscriptionUnverified},
		{name: "Polar is not configured", unconfigured: true, err: billingmodels.ErrOrganizationSubscriptionUnverified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			org := setupBillingOrganization(t)
			require.NoError(t, billingmodels.SetOrganizationPolarCustomerID(database.Conn(), org.ID, "cust_legacy"))
			require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
			usePolarHandler(t, echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
				if tc.fails {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				payload := checkoutPayload("chk_legacy", "open")
				payload["product_id"] = "prod_business"
				delete(payload, "metadata")
				switch r.URL.Path {
				case "/checkouts/":
					assert.Equal(t, org.ID.String(), r.URL.Query().Get("external_customer_id"))
					items := []any{}
					if tc.status != "" {
						items = append(items, payload)
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
						"items": items, "pagination": map[string]int{"max_page": 1},
					}))
				case "/checkouts/chk_legacy":
					payload["status"] = tc.status
					require.NoError(t, json.NewEncoder(w).Encode(payload))
				default:
					t.Errorf("unexpected Polar request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			if tc.unconfigured {
				t.Setenv("POLAR_ACCESS_TOKEN", "")
			}

			clearance, err := VerifyOrganizationDeletion(t.Context(), org.ID)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				assert.Nil(t, clearance)
				if tc.status != "" {
					intent := unresolvedCheckout(t, org)
					require.NotNil(t, intent)
					assert.Equal(t, "chk_legacy", intent.PolarCheckoutIDValue())
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, org.SoftDelete(database.DB(t.Context()), clearance))
			_, err = models.FindOrganizationByID(org.ID.String())
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		})
	}
}
