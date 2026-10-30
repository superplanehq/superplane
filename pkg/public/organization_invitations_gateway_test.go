package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestInvitationGatewayPermissionsAndCapacityDetails(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)
	// Register gateway routes before the catch-all web routes, as the server does.
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	plan := billingmodels.BillingPlan{Key: "invite_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: gatewaySeatCount(50), PolarProductID: new("prod_" + uuid.NewString()), CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ID: "sub_" + uuid.NewString(), ProductID: *plan.PolarProductID,
		Status: billingmodels.PolarSubscriptionStatusActive, PurchasedSeats: new(1),
	}, nil)
	require.NoError(t, err)
	path := "/api/v1/organizations/" + r.Organization.ID.String() + "/invitations"
	response := execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(`{"email":"recipient@example.com"}`)})
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	var body struct {
		Details []struct {
			Type   string `json:"@type"`
			Reason string `json:"reason"`
		} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Len(t, body.Details, 1)
	require.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", body.Details[0].Type)
	require.Equal(t, "SEAT_PURCHASE_REQUIRED", body.Details[0].Reason)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 3).Error)
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(`{"email":"recipient@example.com"}`)})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var created struct {
		Invitation struct {
			ID string `json:"id"`
		} `json:"invitation"`
		AcceptanceURL string `json:"acceptanceUrl"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	require.NotEmpty(t, created.Invitation.ID)
	require.Contains(t, created.AcceptanceURL, "http://localhost/invitations/")
	response = execRequest(server, requestParams{method: http.MethodGet, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "recipient@example.com")
	require.NotContains(t, response.Body.String(), "token")
	require.NotContains(t, response.Body.String(), "acceptance")

	// Use a separate member without member-management permissions.
	account, err := models.CreateAccount("Restricted member", "restricted@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	token, err = authentication.GenerateAccountToken(server.jwt, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, path, `{"email":"second@example.com"}`},
		{http.MethodGet, path, ""},
		{http.MethodPost, path + "/" + created.Invitation.ID + "/resend", "{}"},
		{http.MethodDelete, path + "/" + created.Invitation.ID, ""},
	} {
		response := execRequest(server, requestParams{method: tc.method, path: tc.path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(tc.body)})
		require.Equal(t, http.StatusNotFound, response.Code, "%s %s: %s", tc.method, tc.path, response.Body.String())
	}
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{Name: "inviter", DomainType: models.DomainTypeOrganization, Permissions: []*authorization.Permission{{Resource: "members", Action: "create"}}}))
	require.NoError(t, r.AuthService.AssignRole(member.ID.String(), "inviter", r.Organization.ID.String(), models.DomainTypeOrganization))
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(`{"email":"second@example.com"}`)})
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 4).Error)
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(`{"email":"second@example.com","role":"org_admin"}`)})
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: []byte(`{"email":"second@example.com","role":"org_operator"}`)})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: http.MethodGet, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}})
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())

	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 6).Error)
	for _, scenario := range []struct {
		name   string
		scopes []string
		email  string
		role   string
		status int
	}{
		{name: "create scope cannot grant Admin", scopes: []string{"members:create"}, email: "scoped-invitation@example.com", role: models.RoleOrgAdmin, status: http.StatusForbidden},
		{name: "create scope can invite Operator", scopes: []string{"members:create"}, email: "scoped-invitation@example.com", role: models.RoleOrgOperator, status: http.StatusOK},
		{name: "update scope can grant Admin", scopes: []string{"members:create", "members:update"}, email: "scoped-admin@example.com", role: models.RoleOrgAdmin, status: http.StatusOK},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			scopedToken, err := server.jwt.GenerateScopedToken(jwt.ScopedTokenClaims{Subject: r.User.String(), OrgID: r.Organization.ID.String(), Purpose: "invitations", Scopes: scenario.scopes}, time.Hour)
			require.NoError(t, err)
			payload, err := json.Marshal(map[string]string{"email": scenario.email, "role": scenario.role})
			require.NoError(t, err)
			response := execRequest(server, requestParams{method: http.MethodPost, path: path, authToken: scopedToken, contentType: "application/json", body: payload})
			require.Equal(t, scenario.status, response.Code, response.Body.String())
		})
	}
}

func gatewaySeatCount(count int) *int { return &count }

func TestInvitationGatewayLimitReasons(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	path := "/api/v1/organizations/" + r.Organization.ID.String() + "/invitations"
	invite := func(email string) *httptest.ResponseRecorder {
		body := []byte(`{"email":"` + email + `"}`)
		return execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: token, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}, contentType: "application/json", body: body})
	}
	requireReason := func(response *httptest.ResponseRecorder, status int, reason string) {
		t.Helper()
		require.Equal(t, status, response.Code, response.Body.String())
		var body struct {
			Details []struct {
				Type   string `json:"@type"`
				Reason string `json:"reason"`
			} `json:"details"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Details, 1)
		require.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", body.Details[0].Type)
		require.Equal(t, reason, body.Details[0].Reason)
	}

	t.Setenv("PENDING_INVITATION_LIMIT", "1")
	require.Equal(t, http.StatusOK, invite("first@example.com").Code)
	requireReason(invite("second@example.com"), http.StatusBadRequest, "PENDING_INVITATION_LIMIT_REACHED")

	t.Setenv("PENDING_INVITATION_LIMIT", "100")
	t.Setenv("INVITATION_ORG_HOURLY_LIMIT", "1")
	requireReason(invite("second@example.com"), http.StatusTooManyRequests, "INVITATION_RATE_LIMITED")
}
