package public

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func setupTestServer(t *testing.T) (*Server, *models.Account, string) {
	r := support.Setup(t)

	// Set test environment variables
	os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
	os.Setenv("BASE_URL", "http://localhost:8000")

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, crypto.NewOIDCVerifier(), "", "", r.AuthService)
	require.NoError(t, err)

	token, err := signer.GenerateWithClaims(r.User.String(), time.Hour, map[string]any{
		"org": r.Organization.ID.String(),
	})

	require.NoError(t, err)

	server.RegisterWebRoutes("")

	return server, r.Account, token
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

func TestServer_Logout(t *testing.T) {
	server, _, token := setupTestServer(t)

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
