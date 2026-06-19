package authentication

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type authConfigResponse struct {
	SignupEnabled               bool `json:"signupEnabled"`
	SignupsBlockedByEnvironment bool `json:"signupsBlockedByEnvironment"`
}

func setupAuthHandler(t *testing.T, blockSignup bool) (*Handler, *support.ResourceRegistry) {
	r := support.Setup(t)
	t.Cleanup(func() { r.Close() })

	signer := jwt.NewSigner("test-secret")
	handler := NewHandler(signer, r.Encryptor, r.AuthService, "test", "/templates", blockSignup, false, false)
	return handler, r
}

func TestHandler_handleAuthConfig(t *testing.T) {
	t.Run("reports environment signup block separately from effective signup status", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, true)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/auth/config", nil)

		handler.handleAuthConfig(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)

		var response authConfigResponse
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
		assert.False(t, response.SignupEnabled)
		assert.True(t, response.SignupsBlockedByEnvironment)
	})

	t.Run("reports admin signup state when environment allows signups", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/auth/config", nil)

		handler.handleAuthConfig(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)

		var response authConfigResponse
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
		assert.True(t, response.SignupEnabled)
		assert.False(t, response.SignupsBlockedByEnvironment)
	})
}

func TestSignupsEnabledFromMetadata(t *testing.T) {
	t.Run("defaults to enabled when metadata cannot be loaded", func(t *testing.T) {
		assert.True(t, signupsEnabledFromMetadata(nil, assert.AnError))
	})

	t.Run("uses stored installation setting", func(t *testing.T) {
		assert.True(t, signupsEnabledFromMetadata(&models.InstallationMetadata{SignupsEnabled: true}, nil))
		assert.False(t, signupsEnabledFromMetadata(&models.InstallationMetadata{SignupsEnabled: false}, nil))
	})
}

func TestHandler_findOrCreateAccountForProvider(t *testing.T) {
	t.Run("should find existing account by provider and update email when changed", func(t *testing.T) {
		handler, r := setupAuthHandler(t, false)

		originalEmail := "original@example.com"
		account, err := models.CreateAccount("Test User", originalEmail)
		require.NoError(t, err)

		provider := &models.AccountProvider{
			AccountID:  account.ID,
			Provider:   "github",
			ProviderID: "12345",
			Username:   "testuser",
			Email:      originalEmail,
			Name:       account.Name,
		}
		err = database.Conn().Create(provider).Error
		require.NoError(t, err)

		user := &models.User{
			OrganizationID: r.Organization.ID,
			AccountID:      &account.ID,
			Email:          &originalEmail,
			Name:           account.Name,
		}
		err = database.Conn().Create(user).Error
		require.NoError(t, err)

		newEmail := "newemail@example.com"
		gothUser := goth.User{
			UserID:   "12345",
			Email:    newEmail,
			Name:     "Test User",
			Provider: "github",
		}

		otherProvider := &models.AccountProvider{
			AccountID:  account.ID,
			Provider:   "google",
			ProviderID: "67890",
			Username:   "testuser2",
			Email:      originalEmail,
			Name:       account.Name,
		}
		err = database.Conn().Create(otherProvider).Error
		require.NoError(t, err)

		resultAccount, wasCreated, err := handler.findOrCreateAccountForProvider(gothUser, false)
		require.NoError(t, err)

		assert.Equal(t, account.ID, resultAccount.ID)
		assert.False(t, wasCreated)
		assert.Equal(t, newEmail, resultAccount.Email)

		var accountFromDB models.Account
		err = database.Conn().Where("id = ?", account.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, newEmail, accountFromDB.Email)

		var userFromDB models.User
		err = database.Conn().Where("id = ?", user.ID).First(&userFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, newEmail, userFromDB.GetEmail())

		var providerFromDB models.AccountProvider
		err = database.Conn().Where("id = ?", provider.ID).First(&providerFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, newEmail, providerFromDB.Email)

		var otherProviderFromDB models.AccountProvider
		err = database.Conn().Where("id = ?", otherProvider.ID).First(&otherProviderFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, originalEmail, otherProviderFromDB.Email, "Other provider should keep original email")
	})

	t.Run("should find existing account by email when provider not found", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)

		email := "test@example.com"
		account, err := models.CreateAccount("Test User", email)
		require.NoError(t, err)

		gothUser := goth.User{
			UserID:   "67890",
			Email:    email,
			Name:     "Test User",
			Provider: "google",
		}

		resultAccount, wasCreated, err := handler.findOrCreateAccountForProvider(gothUser, false)
		require.NoError(t, err)

		assert.Equal(t, account.ID, resultAccount.ID)
		assert.False(t, wasCreated)
		assert.Equal(t, email, resultAccount.Email)
	})

	t.Run("should create new account when not found and signup allowed", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)

		gothUser := goth.User{
			UserID:   "99999",
			Email:    "newuser@example.com",
			Name:     "New User",
			Provider: "github",
		}

		resultAccount, wasCreated, err := handler.findOrCreateAccountForProvider(gothUser, true)
		require.NoError(t, err)

		assert.NotNil(t, resultAccount)
		assert.True(t, wasCreated)
		assert.Equal(t, gothUser.Email, resultAccount.Email)
		assert.Equal(t, gothUser.Name, resultAccount.Name)

		var accountFromDB models.Account
		err = database.Conn().Where("id = ?", resultAccount.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, gothUser.Email, accountFromDB.Email)
	})

	t.Run("should return error when signup intent is missing and account not found", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, true)

		gothUser := goth.User{
			UserID:   "88888",
			Email:    "blocked@example.com",
			Name:     "Blocked User",
			Provider: "github",
		}

		resultAccount, wasCreated, err := handler.findOrCreateAccountForProvider(gothUser, false)
		require.Error(t, err)
		assert.Equal(t, errSignupRequired.Error(), err.Error())
		assert.Nil(t, resultAccount)
		assert.False(t, wasCreated)
	})
}

func TestGetRedirectURL(t *testing.T) {
	t.Run("should return home page when no redirect parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/", redirectURL)
	})

	t.Run("should return redirect URL from redirect parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login?redirect=%2Fcanvases", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/canvases", redirectURL)
	})

	t.Run("should return redirect URL from state parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/callback?state=%2Fcanvases%2F123", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/canvases/123", redirectURL)
	})

	t.Run("should return redirect URL from signup state parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/callback?state=signup%3A%252Fcanvases%252F123", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/canvases/123", redirectURL)
	})

	t.Run("should reject absolute URLs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login?redirect=http%3A//evil.com", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/", redirectURL)
	})

	t.Run("should reject protocol-relative URLs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/login?redirect=%2F%2Fevil.com", nil)

		redirectURL := getRedirectURL(req)

		assert.Equal(t, "/", redirectURL)
	})
}

func TestGetSignupRequiredRedirectURL(t *testing.T) {
	t.Run("should redirect to signup with an auth error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/github/callback", nil)

		redirectURL := getSignupRequiredRedirectURL(req)

		assert.Equal(t, "/signup?auth_error=signup_required", redirectURL)
	})

	t.Run("should preserve the original OAuth redirect", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/github/callback?state=%2Finvite%2Fabc", nil)

		redirectURL := getSignupRequiredRedirectURL(req)

		assert.Equal(t, "/signup?auth_error=signup_required&redirect=%2Finvite%2Fabc", redirectURL)
	})
}

func TestHandler_checkSignupPolicy(t *testing.T) {
	t.Run("should reject new magic-code account from login flow", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		req, _ := http.NewRequest("POST", "/auth/magic-code/verify", nil)

		err := handler.checkSignupPolicy("new-magic-code-login@example.com", req)

		assert.Equal(t, errSignupRequired, err)
	})

	t.Run("should allow new magic-code account from signup flow", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		req, _ := http.NewRequest("POST", "/auth/magic-code/verify?signup=true", nil)

		err := handler.checkSignupPolicy("new-magic-code-signup@example.com", req)

		assert.NoError(t, err)
	})

	t.Run("should allow existing account from login flow", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		account, err := models.CreateAccount("Existing User", "existing-magic-code@example.com")
		require.NoError(t, err)
		req, _ := http.NewRequest("POST", "/auth/magic-code/verify", nil)

		err = handler.checkSignupPolicy(account.Email, req)

		assert.NoError(t, err)
	})

	t.Run("should reject new account when installation signups are disabled", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		metadata, err := models.GetInstallationMetadata(database.Conn())
		require.NoError(t, err)
		metadata.SignupsEnabled = false
		metadata.UpdatedAt = time.Now()
		require.NoError(t, models.UpdateInstallationMetadata(database.Conn(), metadata))

		req, _ := http.NewRequest("POST", "/auth/magic-code/verify?signup=true", nil)

		err = handler.checkSignupPolicy("disabled-signup@example.com", req)

		assert.Equal(t, errSignupDisabled, err)
	})
}

func TestGetAuthState(t *testing.T) {
	t.Run("should preserve signup intent and redirect", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/github?signup=true&redirect=%2Finvite%2Fabc", nil)

		state := getAuthState(req)

		assert.Equal(t, "signup:%2Finvite%2Fabc", state)
	})

	t.Run("should preserve login redirect without signup intent", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/auth/github?redirect=%2Finvite%2Fabc", nil)

		state := getAuthState(req)

		assert.Equal(t, "/invite/abc", state)
	})
}

func TestHandler_getPostAuthRedirectURL(t *testing.T) {
	t.Run("should keep redirect for existing users", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, true)
		req, _ := http.NewRequest("GET", "/login?redirect=%2Finvite%2Fabc", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, false)

		assert.Equal(t, "/invite/abc", redirectURL)
	})

	t.Run("should keep redirect when magic code auth is disabled", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, false)
		req, _ := http.NewRequest("GET", "/login?redirect=%2Finvite%2Fabc", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, true)

		assert.Equal(t, "/invite/abc?auth_signup_result=created", redirectURL)
	})

	t.Run("should mark existing user when signup intent resolves to login", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, true)
		req, _ := http.NewRequest("GET", "/callback?state=signup%3A%252Fcanvases", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, false)

		assert.Equal(t, "/canvases?auth_signup_result=existing", redirectURL)
	})

	t.Run("should mark new users when welcome is disabled", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, false)
		req, _ := http.NewRequest("GET", "/callback?state=signup%3A%252Fcanvases%253Fview%253Dlist", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, true)

		assert.Equal(t, "/canvases?auth_signup_result=created&view=list", redirectURL)
	})

	t.Run("should route new cloud users through welcome with original redirect", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, true)
		req, _ := http.NewRequest("GET", "/login?redirect=%2Finvite%2Fabc", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, true)

		assert.Equal(t, "/welcome?redirect=%2Finvite%2Fabc", redirectURL)
	})

	t.Run("should route new cloud users through welcome without empty redirect", func(t *testing.T) {
		handler := NewHandler(nil, nil, nil, "test", "/templates", false, false, true)
		req, _ := http.NewRequest("GET", "/login", nil)

		redirectURL := handler.getPostAuthRedirectURL(req, true)

		assert.Equal(t, "/welcome", redirectURL)
	})
}

func TestWritePostAuthRedirect(t *testing.T) {
	t.Run("should return JSON redirect when requested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/magic-code/verify", nil)
		req.Header.Set("Accept", "application/json")
		recorder := httptest.NewRecorder()

		writePostAuthRedirect(recorder, req, "/welcome?redirect=%2Fcanvases")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"redirectUrl":"/welcome?redirect=%2Fcanvases"}`, recorder.Body.String())
	})

	t.Run("should keep browser redirects by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/magic-code/verify", nil)
		recorder := httptest.NewRecorder()

		writePostAuthRedirect(recorder, req, "/welcome")

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "/welcome", recorder.Header().Get("Location"))
	})
}

func TestHandler_handlePasswordSignup(t *testing.T) {
	t.Run("should route new cloud users through welcome with original redirect", func(t *testing.T) {
		r := support.Setup(t)
		t.Cleanup(func() { r.Close() })

		signer := jwt.NewSigner("test-secret")
		handler := NewHandler(signer, r.Encryptor, r.AuthService, "test", "/templates", false, true, true)

		form := url.Values{}
		form.Set("name", "New User")
		form.Set("email", "new-password-user@example.com")
		form.Set("password", "password123")

		req := httptest.NewRequest(http.MethodPost, "/signup?redirect=%2Fcanvases", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()

		handler.handlePasswordSignup(recorder, req)

		assert.Equal(t, http.StatusSeeOther, recorder.Code)
		assert.Equal(t, "/welcome?redirect=%2Fcanvases", recorder.Header().Get("Location"))
	})
}
