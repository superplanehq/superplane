package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func TestSuspendedOrganizationRetainsReadAndCancelAccess(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	signer := jwt.NewSigner("suspension-test")
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	bearer, err := signer.GenerateScopedToken(jwt.ScopedTokenClaims{Subject: r.User.String(), OrgID: r.Organization.ID.String(), Purpose: "agent-builder", Scopes: []string{"canvases:read", "canvases:create"}}, time.Minute)
	require.NoError(t, err)
	require.NoError(t, r.Organization.Suspend(database.DB(t.Context()), r.Account.ID, "Unattended automation"))
	handler := OrganizationAuthMiddleware(signer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, credential := range []string{"cookie", "bearer"} {
		for _, test := range []struct {
			method, path string
			status       int
		}{
			{"GET", "/api/v1/canvases", 204},
			{"POST", "/api/v1/canvases", 403},
			{"PATCH", "/api/v1/canvases/one", 403},
			{"DELETE", "/api/v1/canvases/one", 403},
			{"POST", "/api/v1/canvases/one/nodes/start/invoke", 403},
			{"PATCH", "/api/v1/canvases/one/runs/two/cancel", 204},
			{"PATCH", "/api/v1/canvases/one/executions/two/cancel", 204},
			{"POST", "/api/v1/agents/chats/one/interrupt", 204},
			{"POST", "/api/v1/canvases/one/node-executions/two/runner-live-logs/session", 204},
		} {
			t.Run(credential+test.method+test.path, func(t *testing.T) {
				request := httptest.NewRequest(test.method, test.path, nil)
				request.Header.Set("x-organization-id", r.Organization.ID.String())
				if credential == "cookie" {
					request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
				} else {
					request.Header.Set("Authorization", "Bearer "+bearer)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				require.Equal(t, test.status, response.Code)
				if test.status == 403 {
					require.Contains(t, response.Body.String(), "ORGANIZATION_SUSPENDED")
				}
			})
		}

	}
}
