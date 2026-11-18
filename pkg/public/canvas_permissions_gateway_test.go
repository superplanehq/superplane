package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasGatewayRejectsUpdateAfterRoleDowngrade(t *testing.T) {
	r := support.Setup(t)
	memberServer, _, _ := setupTestServer(r, t)
	// Register gateway routes before the catch-all web routes, as the server does.
	memberServer.InitRouter()
	registerTestGRPCGateway(t, memberServer, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())

	account, err := models.CreateAccount("Canvas member", "canvas-member@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(member.ID.String(), models.RoleOrgMaintainer, r.Organization.ID.String(), models.DomainTypeOrganization))
	memberToken, err := authentication.GenerateAccountToken(memberServer.jwt, account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	path := "/api/v1/canvases/" + canvas.ID.String()
	headers := map[string]string{"x-organization-id": r.Organization.ID.String()}

	response := execRequest(memberServer, requestParams{
		method: http.MethodPut, path: path, authCookie: memberToken, headers: headers,
		contentType: "application/json", body: []byte(`{"name":"Authorized canvas","description":"Authorized description"}`),
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assertAuthorizedMetadata := func() {
		t.Helper()
		persisted, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		require.Equal(t, "Authorized canvas", persisted.Name)
		require.Equal(t, "Authorized description", persisted.Description)
	}
	assertAuthorizedMetadata()

	// A separate API instance commits the downgrade while the member gateway stays alive.
	adminResources := *r
	adminResources.AuthService = support.AuthService(t)
	adminServer, _, adminToken := setupTestServer(&adminResources, t)
	adminServer.InitRouter()
	registerTestGRPCGateway(t, adminServer, adminResources.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	payload, err := json.Marshal(map[string]string{
		"domainType": "DOMAIN_TYPE_ORGANIZATION", "domainId": r.Organization.ID.String(), "userId": member.ID.String(),
	})
	require.NoError(t, err)
	response = execRequest(adminServer, requestParams{
		method: http.MethodPost, path: "/api/v1/roles/org_operator/users", authCookie: adminToken, headers: headers,
		contentType: "application/json", body: payload,
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	allowed, err := r.AuthService.CheckOrganizationPermission(t.Context(), member.ID.String(), r.Organization.ID.String(), "canvases", "update")
	require.NoError(t, err)
	require.False(t, allowed)

	response = execRequest(memberServer, requestParams{
		method: http.MethodPut, path: path, authCookie: memberToken, headers: headers,
		contentType: "application/json", body: []byte(`{"name":"Revoked canvas","description":"Revoked description"}`),
	})
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	assertAuthorizedMetadata()
	response = execRequest(memberServer, requestParams{method: http.MethodGet, path: path, authCookie: memberToken, headers: headers})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
}
