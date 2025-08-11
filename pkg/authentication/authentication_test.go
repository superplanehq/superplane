package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func setupTestAuth(t *testing.T) (*Handler, *mux.Router) {
	require.NoError(t, database.TruncateTables())

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	authService.EnableCache(false)

	signer := jwt.NewSigner("test-client-secret")
	handler := NewHandler(signer, crypto.NewNoOpEncryptor(), authService, "")

	// Setup test providers
	providers := map[string]ProviderConfig{
		"github": {
			Key:         "test-github-key",
			Secret:      "test-github-secret",
			CallbackURL: "http://localhost:8000/auth/github/callback",
		},
	}
	handler.InitializeProviders(providers)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return handler, router
}

func createTestUser(t *testing.T) *models.User {
	user, err := models.CreateUser(uuid.New(), uuid.New(), "test@example.com", "Test User")
	require.NoError(t, err)
	return user
}

func TestHandler_Login(t *testing.T) {
	_, router := setupTestAuth(t)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Superplane")
	assert.Contains(t, w.Body.String(), "Continue with GitHub")
}

func TestHandler_Logout(t *testing.T) {
	handler, router := setupTestAuth(t)
	user := createTestUser(t)

	token, err := handler.jwtSigner.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

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

func TestHandler_Middleware(t *testing.T) {
	handler, _ := setupTestAuth(t)
	user := createTestUser(t)

	token, err := handler.jwtSigner.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Error("User not found in context")
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	protectedHandler := handler.Middleware(testHandler)

	t.Run("with valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
		})
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})

	t.Run("with invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "invalid-token",
		})
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	})

	t.Run("with bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
