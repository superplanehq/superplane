package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func setupTestServer(t *testing.T) (*Server, *models.User, string) {
	require.NoError(t, database.TruncateTables())

	// Set test environment variables
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("BASE_URL", "http://localhost:8000")

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(&crypto.NoOpEncryptor{}, signer, "", "")
	require.NoError(t, err)

	// Create test user
	user := &models.User{
		Name: "Test User",
	}
	require.NoError(t, user.Create())

	// Create test repo host account
	account := &models.AccountProvider{
		UserID:      user.ID,
		Provider:    "github",
		ProviderID:  "12345",
		Username:    "testuser",
		Email:       "test@example.com",
		Name:        "Test User",
		AccessToken: "encrypted-token",
	}
	require.NoError(t, account.Create())

	// Generate auth token
	token, err := signer.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	server.RegisterWebRoutes("")

	return server, user, token
}

func TestServer_LoginPage(t *testing.T) {
	server, _, _ := setupTestServer(t)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/login",
	})

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "Superplane")
	assert.Contains(t, response.Body.String(), "Continue with GitHub")
}

func TestServer_AuthMe_Unauthorized(t *testing.T) {
	server, _, _ := setupTestServer(t)

	response := execRequest(server, requestParams{
		method: "GET",
		path:   "/auth/me",
	})

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestServer_AuthMe_WithToken(t *testing.T) {
	server, user, token := setupTestServer(t)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var authUser authentication.User
	err := json.Unmarshal(w.Body.Bytes(), &authUser)
	require.NoError(t, err)

	assert.Equal(t, user.Name, authUser.Name)
	assert.Len(t, authUser.AccountProviders, 1)
}

func TestServer_UserProfile_Protected(t *testing.T) {
	server, user, token := setupTestServer(t)

	t.Run("without auth returns unauthorized", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/api/v1/user/profile",
		})

		assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
	})

	t.Run("with valid auth returns profile", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/user/profile", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})

		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var profile models.User
		err := json.Unmarshal(w.Body.Bytes(), &profile)
		require.NoError(t, err)

		assert.Equal(t, user.Name, profile.Name)
	})
}

func TestServer_UserAccountProviders_Protected(t *testing.T) {
	server, _, token := setupTestServer(t)

	t.Run("without auth returns unauthorized", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method: "GET",
			path:   "/api/v1/user/account-providers",
		})

		assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
	})

	t.Run("with valid auth returns accounts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/user/account-providers", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})

		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var accounts []models.AccountProvider
		err := json.Unmarshal(w.Body.Bytes(), &accounts)
		require.NoError(t, err)

		assert.Len(t, accounts, 1)
		assert.Equal(t, "github", accounts[0].Provider)
		assert.Equal(t, "testuser", accounts[0].Username)
	})
}

func TestServer_Logout(t *testing.T) {
	server, _, token := setupTestServer(t)

	req := httptest.NewRequest("GET", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	// Check that auth cookie was cleared
	cookies := w.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie)
	assert.Equal(t, "", authCookie.Value)
	assert.Equal(t, -1, authCookie.MaxAge)
}

func TestServer_DisconnectProvider(t *testing.T) {
	server, _, token := setupTestServer(t)

	req := httptest.NewRequest("POST", "/auth/github/disconnect", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["message"], "Successfully disconnected github account")
}

func TestServer_ProviderConfiguration(t *testing.T) {
	// Test with no providers configured
	os.Unsetenv("GITHUB_CLIENT_ID")
	os.Unsetenv("GITHUB_CLIENT_SECRET")

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
}

func TestServer_WebSocketAuth(t *testing.T) {
	server, _, token := setupTestServer(t)

	canvasID := uuid.New().String()

	t.Run("websocket without auth fails", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws/"+canvasID, nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")

		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})

	t.Run("websocket with auth allows connection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ws/"+canvasID, nil)
		req.Header.Set("Connection", "upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "test-client-id")
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})

		w := httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)

		// WebSocket upgrade will fail in tests, but auth should pass
		// The error will be about websocket upgrade, not auth
		assert.NotEqual(t, http.StatusTemporaryRedirect, w.Code)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
	})
}
