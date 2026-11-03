package public

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/test/support"
)

func TestLegacyAgentEndpointsAreNotRegistered(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	signer := jwt.NewSigner("legacy-route-test")
	oidcProvider := support.NewOIDCProvider()
	server, err := NewServer(r.Encryptor, r.Registry, signer, oidcProvider, "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, oidcProvider)
	server.RegisterWebSocketRoutes()
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/agents/canvases/" + canvas.ID.String() + "/chat"},
		{http.MethodPost, "/api/v1/agents/canvases/" + canvas.ID.String() + "/chat/reset"},
		{http.MethodGet, "/api/v1/agents/chats/chat/messages"},
		{http.MethodPost, "/api/v1/agents/chats/chat/messages"},
		{http.MethodPost, "/api/v1/agents/chats/chat/interrupt"},
		{http.MethodPost, "/api/v1/agents/chats/chat/outcome"},
		{http.MethodGet, "/api/v1/agents/chats/chat/messages/message/images/0"},
		{http.MethodGet, "/ws/agents/sessions/session"},
	} {
		t.Run(endpoint.method+endpoint.path, func(t *testing.T) {
			request := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
			request.Header.Set("X-Organization-ID", r.Organization.ID.String())
			// A handler returning an ownership or feature error is not removal.
			require.False(t, server.Router.Match(request, &mux.RouteMatch{}))
			response := httptest.NewRecorder()
			server.Router.ServeHTTP(response, request)
			require.Equal(t, http.StatusNotFound, response.Code)
		})
	}
}
