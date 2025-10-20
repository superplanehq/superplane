package public

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func setupTestServer(r *support.ResourceRegistry, t *testing.T) (*Server, *models.Account, string) {
	// Set test environment variables
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GOOGLE_CLIENT_ID", "test-google-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-google-client-secret")
	os.Setenv("BASE_URL", "http://localhost:8000")

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, crypto.NewOIDCVerifier(), "", "", "/app/templates", r.AuthService)
	require.NoError(t, err)

	token, err := signer.Generate(r.Account.ID.String(), time.Hour)
	require.NoError(t, err)

	server.RegisterWebRoutes("")

	return server, r.Account, token
}

func Test__Login(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/login",
	})

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "Superplane")
	assert.Contains(t, response.Body.String(), "Continue with GitHub")
	assert.Contains(t, response.Body.String(), "Continue with Google")
}

func Test__Logout(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	req := httptest.NewRequest("GET", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "account_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// Check that auth cookie was cleared
	cookies := w.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "account_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie)
	assert.Equal(t, "", authCookie.Value)
	assert.Equal(t, -1, authCookie.MaxAge)
}

func Test__GetAccount(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	t.Run("no authenticated account -> unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
		assert.Equal(t, "/login", response.Header().Get("Location"))
	})

	t.Run("authenticated account -> authorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func Test__ListAccountOrganizations(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	t.Run("no authenticated account -> unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/organizations", nil)
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
		assert.Equal(t, "/login", response.Header().Get("Location"))
	})

	t.Run("authenticated account -> authorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/organizations", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func Test__CanvasWebSocket(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	t.Run("no authenticated account -> unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/ws/"+uuid.NewString(), nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")

		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("no organization ID header -> unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/ws/"+uuid.NewString(), nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})

		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("canvas that does not exist", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/ws/"+uuid.NewString(), nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.ID.String())

		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("user does not have access to canvas", func(t *testing.T) {
		user := support.CreateUser(t, r, r.Organization.ID)
		canvas := support.CreateCanvas(t, r, r.Organization.ID, user.ID)

		req, _ := http.NewRequest(http.MethodGet, "/ws/"+canvas.ID.String(), nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		req.Header.Set("x-organization-id", r.Organization.ID.String())

		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})
}

func TestServer_ProviderConfiguration(t *testing.T) {
	// Test with no providers configured
	os.Unsetenv("GITHUB_CLIENT_ID")
	os.Unsetenv("GITHUB_CLIENT_SECRET")
	os.Unsetenv("GOOGLE_CLIENT_ID")
	os.Unsetenv("GOOGLE_CLIENT_SECRET")

	providers := getOAuthProviders()
	assert.Empty(t, providers)

	// Test with GitHub configured
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("BASE_URL", "http://localhost:8000")

	providers = getOAuthProviders()
	assert.Contains(t, providers, "github")
	assert.Equal(t, "test-client-id", providers["github"].Key)
	assert.Equal(t, "test-client-secret", providers["github"].Secret)
	assert.Equal(t, "http://localhost:8000/auth/github/callback", providers["github"].CallbackURL)

	// Test with Google configured
	os.Setenv("GOOGLE_CLIENT_ID", "test-google-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-google-client-secret")

	providers = getOAuthProviders()
	assert.Contains(t, providers, "google")
	assert.Equal(t, "test-google-client-id", providers["google"].Key)
	assert.Equal(t, "test-google-client-secret", providers["google"].Secret)
	assert.Equal(t, "http://localhost:8000/auth/google/callback", providers["google"].CallbackURL)

	// Test with both providers configured
	assert.Contains(t, providers, "github")
	assert.Contains(t, providers, "google")
	assert.Len(t, providers, 2)
}
