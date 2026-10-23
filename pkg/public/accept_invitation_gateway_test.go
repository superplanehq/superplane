package public

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAcceptInvitationGatewayAuthenticatesRecipientWithoutMembership(t *testing.T) {
	r := support.Setup(t)
	server, _, ownerCookie := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	t.Setenv(models.BillingEnabledEnv, "no")
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
