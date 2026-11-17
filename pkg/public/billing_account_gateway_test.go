package public

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestBillingGatewayRejectsAccountSpoofing(t *testing.T) {
	r := support.Setup(t)
	server, _, cookie := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	// These are disposable fixtures, including the human's unused eligibility.
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
	require.NoError(t, db.Model(r.Account).Update("welcome_credit_granted_at", nil).Error)
	unrelated, err := models.CreateAccountInTransaction(db, "Unrelated", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	key, err := models.CreateAPIKey(db, r.Organization.ID, "billing-admin", nil, r.User, nil, nil)
	require.NoError(t, err)
	rawToken, err := crypto.Base64String(32)
	require.NoError(t, err)
	require.NoError(t, key.UpdateTokenHash(crypto.HashToken(rawToken)))
	require.NoError(t, r.AuthService.AssignRole(key.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization))
	plan := billingmodels.BillingPlan{Key: "identity_" + uuid.NewString()[:8], Name: "Identity", Category: "paid", Visibility: "public", RequirePaymentMethod: true, TrialCreditCents: 1000, TrialLengthDays: 365, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	cleanupDB := database.DB(context.WithoutCancel(t.Context()))
	t.Cleanup(func() {
		require.NoError(t, cleanupDB.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
		require.NoError(t, cleanupDB.Where("plan_id = ?", plan.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, cleanupDB.Delete(&plan).Error)
		require.NoError(t, cleanupDB.Delete(key).Error)
		require.NoError(t, cleanupDB.Delete(unrelated).Error)
	})
	providerCalls := 0
	polarServer := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
		providerCalls++
		require.Equal(t, http.MethodGet, req.Method)
		items := []map[string]any{}
		switch req.URL.Path {
		case "/checkouts/":
		case "/customers/external/" + r.Organization.ID.String() + "/payment-methods":
			items = append(items, map[string]any{"id": "pm_saved", "type": "card", "method_metadata": map[string]int{"exp_month": 12, "exp_year": time.Now().Year() + 1}})
		default:
			t.Errorf("unexpected Polar request: %s", req.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}}))
	})
	polartest.UseServer(t, polarServer)
	before, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]string{"planKey": plan.Key})
	require.NoError(t, err)
	path := "/api/v1/organizations/" + r.Organization.ID.String() + "/billing/select-plan"
	for _, spoof := range []string{unrelated.ID.String(), ""} {
		headers := map[string]string{}
		if spoof != "" {
			headers["X-Account-Id"] = spoof
		}
		response := execRequest(server, requestParams{method: http.MethodPost, path: path, authToken: rawToken, contentType: "application/json", body: body, headers: headers})
		require.NoError(t, db.First(unrelated, "id = ?", unrelated.ID).Error)
		require.Nil(t, unrelated.WelcomeCreditGrantedAt, "API keys must not consume an unrelated account's eligibility")
		require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
		current, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
		require.NoError(t, err)
		require.Equal(t, before.PlanID, current.PlanID)
		credit, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
		require.NoError(t, err)
		require.Zero(t, credit.GrantMicros)
		var grantCount int64
		require.NoError(t, db.Model(&billingmodels.OrganizationLLMCreditGrant{}).Where("organization_id = ?", r.Organization.ID).Count(&grantCount).Error)
		require.Zero(t, grantCount)
	}
	require.Zero(t, providerCalls, "missing identity must be rejected before provider calls")
	response := execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: cookie, contentType: "application/json", body: body, headers: map[string]string{"X-Organization-Id": r.Organization.ID.String(), "X-Account-Id": unrelated.ID.String(), "Grpc-Metadata-X-Account-Id": unrelated.ID.String()}})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	current, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, current.PlanID)
	require.NoError(t, db.First(unrelated, "id = ?", unrelated.ID).Error)
	require.Nil(t, unrelated.WelcomeCreditGrantedAt)
	require.NoError(t, db.First(r.Account, "id = ?", r.Account.ID).Error)
	require.NotNil(t, r.Account.WelcomeCreditGrantedAt)
	var grants []billingmodels.OrganizationLLMCreditGrant
	require.NoError(t, db.Where("organization_id = ? AND kind = ?", r.Organization.ID, billingmodels.LLMCreditGrantKindWelcome).Find(&grants).Error)
	require.Len(t, grants, 1)
	require.Equal(t, &r.Account.ID, grants[0].ActorAccountID)
	require.Equal(t, billingmodels.CentsToMicros(1000), grants[0].AmountMicros)
}
