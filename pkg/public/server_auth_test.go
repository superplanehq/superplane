package public

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/markbates/goth"
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
	server, err := NewServer(r.Encryptor, r.Registry, signer, crypto.NewOIDCVerifier(), "", "", "", "/app/templates", r.AuthService, false)
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
	assert.Contains(t, strings.ToLower(response.Body.String()), "superplane")
	assert.Contains(t, response.Body.String(), "GitHub")
	assert.Contains(t, response.Body.String(), "Google")
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
		assert.Equal(t, "/login?redirect=%2Faccount", response.Header().Get("Location"))
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
		assert.Equal(t, "/login?redirect=%2Forganizations", response.Header().Get("Location"))
	})

	t.Run("authenticated account -> authorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/organizations", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, req)
		assert.Equal(t, http.StatusOK, response.Code)
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

func TestServer_AuthIntegration(t *testing.T) {
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)

	t.Run("should block signup when configured", func(t *testing.T) {

		signer := jwt.NewSigner("test-client-secret")
		blockedServer, err := NewServer(r.Encryptor, r.Registry, signer, crypto.NewOIDCVerifier(), "", "localhost", "", "/app/templates", r.AuthService, true)
		require.NoError(t, err)

		handler := blockedServer.authHandler
		gothUser := goth.User{
			UserID:   "99999",
			Email:    "newuser@example.com",
			Name:     "New User",
			Provider: "github",
		}

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.Error(t, err)
		assert.Equal(t, "signup is currently disabled", err.Error())
		assert.Nil(t, resultAccount)
	})

	t.Run("should create new account when signup allowed", func(t *testing.T) {
		handler := server.authHandler
		gothUser := goth.User{
			UserID:   "88888",
			Email:    "brandnew@example.com",
			Name:     "Brand New User",
			Provider: "google",
		}

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.NoError(t, err)

		assert.NotNil(t, resultAccount)
		assert.Equal(t, gothUser.Email, resultAccount.Email)
		assert.Equal(t, gothUser.Name, resultAccount.Name)
	})
}
