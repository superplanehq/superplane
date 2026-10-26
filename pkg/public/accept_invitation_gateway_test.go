package public

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestAcceptInvitationGatewayAuthenticatesRecipientWithoutMembership(t *testing.T) {
	r := support.Setup(t)
	server, _, ownerCookie := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	token := "recipient-invitation"
	hash := crypto.HashToken(token)
	invitation := models.OrganizationInvitation{OrganizationID: r.Organization.ID, Email: account.Email, Role: models.RoleOrgMaintainer, InvitedBy: r.User, State: models.InvitationStatePending, TokenHash: &hash, SentAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, invitation.Save(database.Conn()))
	path := "/api/v1/invitations/" + token + "/accept"
	response := execRequest(server, requestParams{method: http.MethodPost, path: path})
	require.Equal(t, http.StatusUnauthorized, response.Code)
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: ownerCookie, headers: map[string]string{"x-account-id": account.ID.String()}})
	require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	var rejected status.Status
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &rejected))
	require.Len(t, rejected.Details, 1)
	info := &errdetails.ErrorInfo{}
	require.NoError(t, rejected.Details[0].UnmarshalTo(info))
	require.Equal(t, "superplane.com", info.Domain)
	require.Equal(t, "INVITATION_EMAIL_MISMATCH", info.Reason)
	require.Equal(t, map[string]string{"invited_email": "i***@example.com"}, info.Metadata)
	cookie, err := authentication.GenerateAccountToken(server.jwt, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	response = execRequest(server, requestParams{method: http.MethodPost, path: path, authCookie: cookie})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var accepted map[string]string
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &accepted))
	require.Equal(t, r.Organization.ID.String(), accepted["organization_id"])
	require.Equal(t, "joined", accepted["status"])
	require.False(t, strings.Contains(response.Body.String(), "token"))
}

func TestInvitationGatewayReportsPastDuePayment(t *testing.T) {
	r := support.Setup(t)
	server, _, ownerCookie := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	productID := "prod_invite_" + uuid.NewString()
	subscriptionID := "sub_" + uuid.NewString()
	plan := billingmodels.BillingPlan{Key: "invite_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: gatewaySeatCount(50), PolarProductID: &productID, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ID: subscriptionID, ProductID: productID, Status: billingmodels.PolarSubscriptionStatusActive, PurchasedSeats: gatewaySeatCount(3),
	}, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
		"polar_subscription_status": billingmodels.PolarSubscriptionStatusPastDue,
	}).Error)

	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	token := "past-due-invitation"
	hash := crypto.HashToken(token)
	invitation := models.OrganizationInvitation{OrganizationID: r.Organization.ID, Email: account.Email, Role: models.RoleOrgOperator, InvitedBy: r.User, State: models.InvitationStatePending, TokenHash: &hash, SentAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, invitation.Save(db))
	cookie, err := authentication.GenerateAccountToken(server.jwt, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	response := execRequest(server, requestParams{method: http.MethodPost, path: "/api/v1/invitations/" + token + "/accept", authCookie: cookie})
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	var rejected status.Status
	require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), &rejected))
	require.Equal(t, int32(codes.FailedPrecondition), rejected.Code)
	require.Equal(t, "This organization cannot add members until its payment is updated. Ask an organization admin.", rejected.Message)
	require.Len(t, rejected.Details, 1)
	info := &errdetails.ErrorInfo{}
	require.NoError(t, rejected.Details[0].UnmarshalTo(info))
	require.Equal(t, "superplane.com", info.Domain)
	require.Equal(t, "PAYMENT_PAST_DUE", info.Reason)

	for _, phase := range []struct {
		status  string
		blocked bool
	}{
		{status: billingmodels.PolarSubscriptionStatusPastDue, blocked: true},
		{status: billingmodels.PolarSubscriptionStatusActive, blocked: false},
	} {
		t.Run(phase.status, func(t *testing.T) {
			_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
				ID: subscriptionID, ProductID: productID, Status: phase.status, PurchasedSeats: gatewaySeatCount(3),
			}, nil)
			require.NoError(t, err)
			response := execRequest(server, requestParams{method: http.MethodGet, path: "/api/v1/organizations/" + r.Organization.ID.String() + "/invitations", authCookie: ownerCookie, headers: map[string]string{"x-organization-id": r.Organization.ID.String()}})
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var listed struct {
				Seats struct {
					PaymentBlocked *bool `json:"paymentBlocked"`
				} `json:"seats"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &listed))
			require.NotNil(t, listed.Seats.PaymentBlocked, response.Body.String())
			require.Equal(t, phase.blocked, *listed.Seats.PaymentBlocked)
		})
	}
}
