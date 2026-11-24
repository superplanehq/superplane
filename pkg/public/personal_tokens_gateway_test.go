package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestPersonalTokenGatewayCredentialEligibility(t *testing.T) {
	r := support.Setup(t)
	server, _, session := setupTestServer(r, t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	db := database.DB(t.Context())
	allowedCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	scopedToken, err := server.jwt.GenerateScopedToken(jwt.ScopedTokenClaims{
		Subject: r.User.String(), OrgID: r.Organization.ID.String(), Purpose: "audit-fixture",
		Scopes: []string{"canvases:read:" + allowedCanvas.ID.String()},
	}, 5*time.Minute)
	require.NoError(t, err)
	broadScopedToken, err := server.jwt.GenerateScopedToken(jwt.ScopedTokenClaims{
		Subject: r.User.String(), OrgID: r.Organization.ID.String(), Purpose: "audit-fixture",
		Scopes: []string{"canvases:read", "canvases:create"},
	}, 5*time.Minute)
	require.NoError(t, err)
	personalBearer := "dummy-personal-bearer"
	seedToken := models.NewUserAPIToken(r.User, "existing-fixture", crypto.HashToken(personalBearer))
	require.NoError(t, models.CreateUserAPIToken(db, seedToken))
	apiKey, err := models.CreateAPIKey(db, r.Organization.ID, "audit-bot", nil, r.User, nil, nil)
	require.NoError(t, err)
	apiBearer := "dummy-api-key-bearer"
	require.NoError(t, apiKey.UpdateTokenHash(crypto.HashToken(apiBearer)))

	request := func(method, path, body, bearer, cookie string, headers map[string]string) *httptest.ResponseRecorder {
		return execRequest(server, requestParams{
			method: method, path: path, body: []byte(body), authToken: bearer, authCookie: cookie,
			contentType: "application/json", headers: headers,
		})
	}
	checkCanvasAccess := func(t *testing.T, bearer string, createStatus, otherStatus int) {
		t.Helper()
		response := request(http.MethodGet, "/api/v1/canvases/"+allowedCanvas.ID.String(), "", bearer, "", nil)
		require.Equal(t, http.StatusOK, response.Code)
		response = request(http.MethodGet, "/api/v1/canvases/"+otherCanvas.ID.String(), "", bearer, "", nil)
		require.Equal(t, otherStatus, response.Code)
		response = request(http.MethodPost, "/api/v1/canvases", `{"name":"audit-`+uuid.NewString()+`"}`, bearer, "", nil)
		require.Equal(t, createStatus, response.Code)
	}
	checkCanvasAccess(t, personalBearer, http.StatusOK, http.StatusOK)
	checkCanvasAccess(t, scopedToken, http.StatusNotFound, http.StatusNotFound)

	for _, endpoint := range []struct{ name, path, body, secretField string }{
		{"named", "/api/v1/me/tokens", `{"name":"audit-fixture"}`, "plaintext"},
		{"legacy", "/api/v1/me/token", "", "token"},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			for _, credential := range []struct {
				name, bearer string
				headers      map[string]string
			}{
				{name: "resource-scoped", bearer: scopedToken},
				{name: "broad-scoped", bearer: broadScopedToken},
				{name: "forged-scope-headers", bearer: scopedToken, headers: map[string]string{
					"X-Token-Scopes": "[]", "Grpc-Metadata-X-Token-Scopes": "[]",
				}},
				{name: "api-key", bearer: apiBearer},
			} {
				t.Run(credential.name, func(t *testing.T) {
					legacyHash := crypto.HashToken("dummy-existing-legacy-token")
					require.NoError(t, r.UserModel.UpdateTokenHash(legacyHash))
					before, err := models.ListUserAPITokens(db, r.User)
					require.NoError(t, err)
					require.NotEmpty(t, before)
					response := request(http.MethodPost, endpoint.path, endpoint.body, credential.bearer, "", credential.headers)
					require.Equal(t, http.StatusForbidden, response.Code)
					var denied struct {
						Code      int             `json:"code"`
						Token     json.RawMessage `json:"token"`
						Plaintext string          `json:"plaintext"`
					}
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &denied))
					require.Equal(t, 7, denied.Code)
					require.Empty(t, denied.Token)
					require.Empty(t, denied.Plaintext)
					after, err := models.ListUserAPITokens(db, r.User)
					require.NoError(t, err)
					require.Equal(t, before, after)
					user, err := models.FindActiveUserByIDAnyOrg(db, r.User)
					require.NoError(t, err)
					require.Equal(t, legacyHash, user.TokenHash)
				})
			}

			for _, credential := range []struct{ name, bearer, cookie string }{
				{"human-session", "", session},
				{"personal-bearer", personalBearer, ""},
			} {
				t.Run(credential.name, func(t *testing.T) {
					// Client scope headers must not turn an unrestricted credential into scoped authentication.
					headers := map[string]string{"X-Organization-Id": r.Organization.ID.String(), "X-Token-Scopes": `["canvases:read"]`, "Grpc-Metadata-X-Token-Scopes": `["canvases:read"]`}
					response := request(http.MethodPost, endpoint.path, endpoint.body, credential.bearer, credential.cookie, headers)
					require.Equal(t, http.StatusOK, response.Code)
					var issued map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &issued))
					var secret string
					require.NoError(t, json.Unmarshal(issued[endpoint.secretField], &secret))
					require.NotEmpty(t, secret)
					response = request(http.MethodGet, "/api/v1/me", "", secret, "", nil)
					require.Equal(t, http.StatusOK, response.Code)
					var me struct {
						User struct {
							ID             string `json:"id"`
							OrganizationID string `json:"organizationId"`
						} `json:"user"`
					}
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &me))
					require.Equal(t, r.User.String(), me.User.ID)
					require.Equal(t, r.Organization.ID.String(), me.User.OrganizationID)
				})
			}
		})
	}
}
