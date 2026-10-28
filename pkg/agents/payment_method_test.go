package agents_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
)

func TestAgentUsageRequiresPlanPaymentMethod(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	provider := &fakeProvider{}
	service := newService(t, r, provider)
	canvas := setupCanvasForUser(t, r)
	session, err := service.EnsureSession(t.Context(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	db := database.DB(t.Context())
	plan := billingmodels.BillingPlan{Key: "agent_card_" + uuid.NewString()[:8], Name: "Card required", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	plan.RequirePaymentMethod = true
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/customers/external/"+r.Organization.ID.String()+"/payment-methods", req.URL.Path)
		_, _ = w.Write([]byte(`{"items":[],"pagination":{"max_page":1}}`))
	}))
	defer server.Close()
	t.Setenv("POLAR_ACCESS_TOKEN", "test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	_, err = service.EnsureSession(t.Context(), r.Organization.ID, r.User, canvas.ID)
	require.ErrorIs(t, err, billingmodels.ErrPaymentMethodRequired)
	_, err = service.ResetSession(t.Context(), r.Organization.ID, r.User, canvas.ID)
	require.ErrorIs(t, err, billingmodels.ErrPaymentMethodRequired)
	require.Equal(t, 1, provider.createCalled)
	_, err = service.SendMessage(t.Context(), r.Organization.ID, r.User, session.ID, "Start work", nil)
	require.ErrorIs(t, err, billingmodels.ErrPaymentMethodRequired)
	require.Zero(t, provider.sendCalled)
	err = service.DefineOutcome(t.Context(), r.Organization.ID, r.User, session.ID, "Build it", "Works", 1)
	require.ErrorIs(t, err, billingmodels.ErrPaymentMethodRequired)
	require.Empty(t, provider.defineSessions)
}

func TestAgentUsageRequiresActivePlan(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	provider := &fakeProvider{}
	service := newService(t, r, provider)
	canvas := setupCanvasForUser(t, r)
	session, err := service.EnsureSession(t.Context(), r.Organization.ID, r.User, canvas.ID)
	require.NoError(t, err)
	db := database.DB(t.Context())
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, billingmodels.BillingPlanNone, nil)
	require.NoError(t, err)
	credit, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	require.Greater(t, credit.RemainingMicros, int64(0), "the org must keep leftover credit")

	_, err = service.EnsureSession(t.Context(), r.Organization.ID, r.User, canvas.ID)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	_, err = service.SendMessage(t.Context(), r.Organization.ID, r.User, session.ID, "Start work", nil)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	require.Zero(t, provider.sendCalled)
	err = service.DefineOutcome(t.Context(), r.Organization.ID, r.User, session.ID, "Build it", "Works", 1)
	require.ErrorIs(t, err, billingmodels.ErrHostedSubscriptionRequired)
	require.Empty(t, provider.defineSessions)
}
