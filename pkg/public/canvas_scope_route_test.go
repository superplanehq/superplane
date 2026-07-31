package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

// Routes registered directly on the router do not pass through
// grpcGatewayHandler or GatewayAuthorizer.AuthorizeHTTP, so the canvas
// restriction on an API key has to be enforced by the handler itself. Without
// it, a key restricted to one canvas can act on every canvas in the
// organization.
func Test__DirectlyRegisteredCanvasRoutes__EnforceAPIKeyCanvasScope(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	registerTestGRPCGateway(
		t, server, r.AuthService, r.Registry, r.Encryptor,
		support.NewOIDCProvider(), r.GitProvider, nil,
	)

	scopedCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	//
	// An API key restricted to scopedCanvas, holding the lowest role that still
	// grants canvases:read organization-wide.
	//
	plainToken, err := crypto.Base64String(64)
	require.NoError(t, err)

	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"scoped-key",
		nil,
		r.User,
		nil,
		[]string{scopedCanvas.ID.String()},
	)
	require.NoError(t, err)

	apiKey.TokenHash = crypto.HashToken(plainToken)
	require.NoError(t, database.Conn().Save(apiKey).Error)
	require.NoError(t, r.AuthService.AssignRole(
		apiKey.ID.String(),
		models.RoleOrgViewer,
		r.Organization.ID.String(),
		models.DomainTypeOrganization,
	))

	paths := map[string]func(canvasID string) string{
		"repository file download": func(canvasID string) string {
			return "/api/v1/canvases/" + canvasID + "/repository/file?path=README.md"
		},
		"runner live log session": func(canvasID string) string {
			return "/api/v1/canvases/" + canvasID + "/node-executions/" +
				"00000000-0000-0000-0000-000000000001/runner-live-logs/session"
		},
	}

	t.Run("canvas websocket handshake is rejected outside the key's scope", func(t *testing.T) {
		server.RegisterWebRoutes("/")

		httpServer := httptest.NewServer(server.Router)
		defer httpServer.Close()

		dialer := websocket.Dialer{}
		header := http.Header{}
		header.Set("Authorization", "Bearer "+plainToken)

		conn, response, err := dialer.Dial(
			"ws"+strings.TrimPrefix(httpServer.URL, "http")+"/ws/"+otherCanvas.ID.String(),
			header,
		)
		if conn != nil {
			conn.Close()
		}

		require.Error(t, err, "handshake must not succeed for a canvas outside the key's scope")
		require.NotNil(t, response)
		require.Equal(t, http.StatusNotFound, response.StatusCode)
	})

	for name, path := range paths {
		t.Run(name+" is forbidden on a canvas outside the key's scope", func(t *testing.T) {
			response := execRequest(server, requestParams{
				method:    "GET",
				path:      path(otherCanvas.ID.String()),
				authToken: plainToken,
			})

			require.Equal(t, http.StatusForbidden, response.Code, "body: %s", response.Body.String())
		})

		t.Run(name+" passes the scope check on the key's own canvas", func(t *testing.T) {
			response := execRequest(server, requestParams{
				method:    "GET",
				path:      path(scopedCanvas.ID.String()),
				authToken: plainToken,
			})

			//
			// The request is expected to fail further in — there is no such file
			// and no such execution — but it must get past the scope check.
			//
			require.NotEqual(t, http.StatusForbidden, response.Code, "body: %s", response.Body.String())
		})
	}
}
