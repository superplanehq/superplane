package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAuthorizeCanvasRead_APIKeyCanvasScope(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
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
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider(), r.GitProvider, nil)

	canvasA, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)
	canvasB, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)

	rawToken, err := crypto.Base64String(32)
	require.NoError(t, err)
	description := "scoped key"
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		r.Organization.ID,
		"scoped-download-key",
		&description,
		r.User,
		nil,
		[]string{canvasA.ID.String()},
	)
	require.NoError(t, err)
	require.NoError(t, apiKey.UpdateTokenHash(crypto.HashToken(rawToken)))
	require.NoError(t, r.AuthService.AssignRole(apiKey.ID.String(), models.RoleOrgViewer, r.Organization.ID.String(), models.DomainTypeOrganization))

	downloadWithAPIKey := func(canvasID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/canvases/%s/repository/file?path=README.md", canvasID),
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+rawToken)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("denies download for canvas outside API key scope", func(t *testing.T) {
		response := downloadWithAPIKey(canvasB.ID.String())
		assert.Equal(t, http.StatusForbidden, response.Code)
	})

	t.Run("allows download for canvas inside API key scope", func(t *testing.T) {
		response := downloadWithAPIKey(canvasA.ID.String())
		assert.NotEqual(t, http.StatusForbidden, response.Code)
		assert.NotEqual(t, http.StatusUnauthorized, response.Code)
	})
}

func TestHandleWebSocket_RequiresCanvasRead(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
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
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider(), r.GitProvider, nil)
	server.RegisterWebRoutes("")

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)

	restrictedAccount, err := models.CreateAccount("ws-restricted@example.com", "WS Restricted")
	require.NoError(t, err)
	_, err = models.CreateUser(r.Organization.ID, restrictedAccount.ID, restrictedAccount.Email, restrictedAccount.Name)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/ws/%s", canvas.ID.String()), nil)
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	token, err := authentication.GenerateAccountToken(signer, restrictedAccount.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
