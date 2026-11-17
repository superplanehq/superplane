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

func TestRepositoryFileDownloadCanvasScopes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, signer := mustRunnerLiveLogServer(t, r)
	canvasA, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	canvasB, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	credentials := canvasReadCredentials(t, r, signer, canvasA.ID)
	// The gateway already enforces the persisted API-key restriction.
	for _, item := range []struct {
		id     uuid.UUID
		status int
	}{{canvasA.ID, http.StatusOK}, {canvasB.ID, http.StatusNotFound}} {
		response := canvasReadGET(server, credentials[0].token, "/api/v1/canvases/"+item.id.String())
		require.Equal(t, item.status, response.Code, response.Body.String())
	}
	for _, credential := range credentials {
		t.Run(credential.name, func(t *testing.T) {
			queries := observeCanvasReadQueries(t)
			for _, item := range []struct {
				canvas  *models.Canvas
				allowed bool
			}{{canvasA, credential.allowedA}, {canvasB, credential.allowedB}} {
				for _, suffix := range []string{"", "&stage=true", "&version_id=" + item.canvas.LiveVersionID.String()} {
					*queries = nil
					response := canvasReadGET(server, credential.token, "/api/v1/canvases/"+item.canvas.ID.String()+"/file?path=canvas.yaml"+suffix)
					if !item.allowed {
						require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
						require.Empty(t, *queries, "denial must precede content reads")
						continue
					}
					require.Equal(t, http.StatusOK, response.Code, response.Body.String())
					require.Contains(t, response.Body.String(), "kind: Canvas")
					require.NotEmpty(t, *queries)
				}
			}
		})
	}
	foreignOrg := support.CreateOrganization(t, r, r.User)
	foreignCanvas, _ := support.CreateCanvas(t, foreignOrg.ID, r.User, nil, nil)
	response := canvasReadGET(server, credentials[len(credentials)-1].token, "/api/v1/canvases/"+foreignCanvas.ID.String()+"/file?path=canvas.yaml")
	require.Equal(t, http.StatusNotFound, response.Code)
}
