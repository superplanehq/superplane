package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func downloadFile(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	organizationID uuid.UUID,
	accountID *uuid.UUID,
	canvasID string,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()

	url := fmt.Sprintf("/api/v1/canvases/%s/file", canvasID)
	if path != "" {
		url += "?path=" + path
	}

	req := httptest.NewRequest(http.MethodGet, url, nil)
	if accountID != nil {
		req.Header.Set("x-organization-id", organizationID.String())
		token, err := authentication.GenerateAccountToken(signer, accountID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	}

	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func Test__RepositoryFileDownload(t *testing.T) {
	r := support.Setup(t)
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService, false,
	)

	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())

	authenticated := &r.Account.ID

	t.Run("missing path -> bad request", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "")
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "path is required")
	})

	t.Run("invalid canvas id -> bad request", func(t *testing.T) {
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, "invalid-id", "canvas.yaml")
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "Invalid canvas_id")
	})

	t.Run("unauthenticated -> unauthorized", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		response := downloadFile(t, server, signer, r.Organization.ID, nil, canvas.ID.String(), "canvas.yaml")
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("user without canvas access -> forbidden", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		restrictedAccount, err := models.CreateAccount("restricted@example.com", "Restricted User")
		require.NoError(t, err)
		_, err = models.CreateUser(r.Organization.ID, restrictedAccount.ID, restrictedAccount.Email, restrictedAccount.Name)
		require.NoError(t, err)

		response := downloadFile(t, server, signer, r.Organization.ID, &restrictedAccount.ID, canvas.ID.String(), "canvas.yaml")
		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Contains(t, response.Body.String(), "Unauthorized")
	})

	t.Run("canvas not found -> not found", func(t *testing.T) {
		invalidID := uuid.NewString()
		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, invalidID, "canvas.yaml")
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "Canvas not found")
	})

	t.Run("non-spec path -> not found", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "README.md")
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "File not found")
	})

	t.Run("canvas.yaml -> generated spec", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		response := downloadFile(t, server, signer, r.Organization.ID, authenticated, canvas.ID.String(), "canvas.yaml")
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), "kind: Canvas")
	})
}
