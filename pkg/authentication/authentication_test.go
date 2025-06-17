package authentication

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func setupTestAuth(t *testing.T) (*Handler, *mux.Router) {
	require.NoError(t, database.TruncateTables())

	signer := jwt.NewSigner("test-client-secret")
	handler := NewHandler(signer, crypto.NewNoOpEncryptor(), "")

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
	user := &models.User{
		Name: "Test User",
	}
	require.NoError(t, user.Create())
	return user
}

func createTestAccountProvider(t *testing.T, userID uuid.UUID) *models.AccountProvider {
	account := &models.AccountProvider{
		UserID:      userID,
		Provider:    "github",
		ProviderID:  "12345",
		Username:    "testuser",
		Email:       "test@example.com",
		Name:        "Test User",
		AccessToken: "encrypted-token",
	}
	require.NoError(t, account.Create())
	return account
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

func TestHandler_Me_Unauthorized(t *testing.T) {
	_, router := setupTestAuth(t)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Me_WithValidToken(t *testing.T) {
	handler, router := setupTestAuth(t)
	user := createTestUser(t)
	createTestAccountProvider(t, user.ID)

	token, err := handler.jwtSigner.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var authUser User
	err = json.Unmarshal(w.Body.Bytes(), &authUser)
	require.NoError(t, err)

	assert.Equal(t, user.Name, authUser.Name)
	assert.Len(t, authUser.AccountProviders, 1)
	assert.Equal(t, "github", authUser.AccountProviders[0].Provider)
}

func TestHandler_DisconnectProvider(t *testing.T) {
	handler, router := setupTestAuth(t)
	user := createTestUser(t)
	account := createTestAccountProvider(t, user.ID)

	token, err := handler.jwtSigner.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/github/disconnect", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify account was deleted
	_, err = models.FindAccountProviderByID(account.ID.String())
	assert.Error(t, err)
}

func TestHandler_DisconnectProvider_NotFound(t *testing.T) {
	handler, router := setupTestAuth(t)
	user := createTestUser(t)

	token, err := handler.jwtSigner.Generate(user.ID.String(), time.Hour)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/auth/github/disconnect", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
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
