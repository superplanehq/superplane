package authentication

import (
	"net/http"
	"testing"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func setupAuthHandler(t *testing.T, blockSignup bool) (*Handler, *support.ResourceRegistry) {
	r := support.Setup(t)
	t.Cleanup(func() { r.Close() })

	signer := jwt.NewSigner("test-secret")
	handler := NewHandler(signer, r.Encryptor, r.AuthService, "test", "/templates", blockSignup, false, false)
	return handler, r
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

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.NoError(t, err)

		assert.Equal(t, account.ID, resultAccount.ID)
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

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.NoError(t, err)

		assert.Equal(t, account.ID, resultAccount.ID)
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

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.NoError(t, err)

		assert.NotNil(t, resultAccount)
		assert.Equal(t, gothUser.Email, resultAccount.Email)
		assert.Equal(t, gothUser.Name, resultAccount.Name)

		var accountFromDB models.Account
		err = database.Conn().Where("id = ?", resultAccount.ID).First(&accountFromDB).Error
		require.NoError(t, err)
		assert.Equal(t, gothUser.Email, accountFromDB.Email)
	})

	t.Run("should return error when signup blocked and account not found", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, true)

		gothUser := goth.User{
			UserID:   "88888",
			Email:    "blocked@example.com",
			Name:     "Blocked User",
			Provider: "github",
		}

		resultAccount, err := handler.FindOrCreateAccountForProvider(gothUser)
		require.Error(t, err)
		assert.Equal(t, SignupDisabledError, err.Error())
		assert.Nil(t, resultAccount)
	})
}

func TestAcceptPendingInvitations(t *testing.T) {
	setupInvitation := func(t *testing.T, email string) (*Handler, *support.ResourceRegistry, *models.Account) {
		t.Helper()
		handler, r := setupAuthHandler(t, false)
		account, err := models.CreateAccount("Test User", email)
		require.NoError(t, err)
		_, err = models.CreateInvitation(r.Organization.ID, r.User, email, models.InvitationStatePending)
		require.NoError(t, err)
		return handler, r, account
	}

	t.Run("should accept invitation when provider is allowed", func(t *testing.T) {
		email := "github-user@example.com"
		handler, r, account := setupInvitation(t, email)

		err := handler.acceptPendingInvitations(account, models.ProviderGitHub)
		require.NoError(t, err)

		_, err = models.FindPendingInvitation(email, r.Organization.ID.String())
		assert.Error(t, err, "invitation should no longer be pending")
	})

	t.Run("should skip invitation when provider is not allowed", func(t *testing.T) {
		email := "google-user@example.com"
		handler, r, account := setupInvitation(t, email)

		err := database.Conn().Model(r.Organization).Update("allowed_providers", datatypes.JSONSlice[string]{models.ProviderGitHub}).Error
		require.NoError(t, err)

		err = handler.acceptPendingInvitations(account, models.ProviderGoogle)
		require.NoError(t, err)

		invitation, err := models.FindPendingInvitation(email, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.InvitationStatePending, invitation.State)
	})

	t.Run("should accept invitation when provider is empty (password/magic-code login)", func(t *testing.T) {
		email := "password-user@example.com"
		handler, r, account := setupInvitation(t, email)

		err := handler.acceptPendingInvitations(account, "")
		require.NoError(t, err)

		_, err = models.FindPendingInvitation(email, r.Organization.ID.String())
		assert.Error(t, err, "invitation should no longer be pending")
	})

	t.Run("should skip invitation when non-OAuth invite completion disabled for org", func(t *testing.T) {
		email := "no-non-oauth-invite@example.com"
		handler, r, account := setupInvitation(t, email)

		err := database.Conn().Model(r.Organization).Update("allow_direct_email_invite_completion", false).Error
		require.NoError(t, err)

		err = handler.acceptPendingInvitations(account, "")
		require.NoError(t, err)

		invitation, err := models.FindPendingInvitation(email, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.InvitationStatePending, invitation.State)
	})

	t.Run("should skip invitation when org no longer exists", func(t *testing.T) {
		email := "deleted-org-user@example.com"
		handler, r, account := setupInvitation(t, email)

		err := models.SoftDeleteOrganization(r.Organization.ID.String())
		require.NoError(t, err)

		err = handler.acceptPendingInvitations(account, models.ProviderGitHub)
		require.NoError(t, err)

		invitation, err := models.FindPendingInvitation(email, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.InvitationStatePending, invitation.State)
	})

	t.Run("should accept invitation when org has no provider restriction", func(t *testing.T) {
		email := "unrestricted-org-user@example.com"
		handler, r, account := setupInvitation(t, email)

		err := database.Conn().Model(r.Organization).Update("allowed_providers", datatypes.JSONSlice[string]{}).Error
		require.NoError(t, err)

		err = handler.acceptPendingInvitations(account, models.ProviderGitHub)
		require.NoError(t, err)

		_, err = models.FindPendingInvitation(email, r.Organization.ID.String())
		assert.Error(t, err, "invitation should no longer be pending")
	})

	t.Run("should accept only invitations from orgs that allow the provider", func(t *testing.T) {
		email := "multi-org-user@example.com"
		handler, r, account := setupInvitation(t, email)

		// Create a second org that only allows Google.
		googleOrg, err := models.CreateOrganization(support.RandomName("org"), support.RandomName("org-display"))
		require.NoError(t, err)
		err = database.Conn().Model(googleOrg).Update("allowed_providers", datatypes.JSONSlice[string]{models.ProviderGoogle}).Error
		require.NoError(t, err)
		_, err = models.CreateInvitation(googleOrg.ID, r.User, email, models.InvitationStatePending)
		require.NoError(t, err)

		err = handler.acceptPendingInvitations(account, models.ProviderGitHub)
		require.NoError(t, err)

		_, err = models.FindPendingInvitation(email, r.Organization.ID.String())
		assert.Error(t, err, "github org invitation should be accepted")

		pending, err := models.FindPendingInvitation(email, googleOrg.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.InvitationStatePending, pending.State)
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
