package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestGetAccount_ListsProviders(t *testing.T) {
	r := support.Setup(t)
	server, account, token := setupTestServer(r, t)

	req, _ := http.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var resp AccountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	assert.Equal(t, account.ID.String(), resp.ID)
	require.NotEmpty(t, resp.Providers)
	assert.Equal(t, models.ProviderGitHub, resp.Providers[0].Provider)
	assert.False(t, resp.HasPassword)
}

func TestUpdateAccount_UpdatesName(t *testing.T) {
	r := support.Setup(t)
	server, account, token := setupTestServer(r, t)

	body, err := json.Marshal(map[string]string{"name": "Ada Byron"})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodPatch, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var resp AccountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	assert.Equal(t, "Ada Byron", resp.Name)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "Ada Byron", reloaded.Name)
}

func TestDisconnectAccountProvider_RefusesLastMethod(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	req, _ := http.NewRequest(http.MethodDelete, "/account/providers/github", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusConflict, res.Code)
	assert.Contains(t, res.Body.String(), "Keep at least one sign-in method.")
}

func TestUpdateAccount_SwitchesEmailFromConnectedProvider(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Create(&models.AccountProvider{
		AccountID:  r.Account.ID,
		Provider:   models.ProviderGoogle,
		ProviderID: "google-ada",
		Email:      "ada-google@example.com",
		Username:   "ada",
		Name:       r.Account.Name,
	}).Error)

	server, account, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": "ada-google@example.com"})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodPatch, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var resp AccountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	assert.Equal(t, "ada-google@example.com", resp.Email)

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "ada-google@example.com", reloaded.Email)
}

func TestUpdateAccount_KeepsPasswordEmailAfterSwitch(t *testing.T) {
	r := support.Setup(t)
	hash, err := crypto.HashPassword("current-pass-123")
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Create(&models.AccountProvider{
		AccountID:  r.Account.ID,
		Provider:   models.ProviderGoogle,
		ProviderID: "google-ada",
		Email:      "ada-google@example.com",
		Username:   "ada",
		Name:       r.Account.Name,
	}).Error)

	originalEmail := r.Account.Email
	server, account, token := setupTestServer(r, t)

	switchToGoogle, err := json.Marshal(map[string]string{"email": "ada-google@example.com"})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodPatch, "/account", bytes.NewReader(switchToGoogle))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)

	switchBack, err := json.Marshal(map[string]string{"email": originalEmail})
	require.NoError(t, err)
	req, _ = http.NewRequest(http.MethodPatch, "/account", bytes.NewReader(switchBack))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res = httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)

	var resp AccountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	assert.Equal(t, originalEmail, resp.Email)
	require.True(t, slices.ContainsFunc(resp.Providers, func(provider AccountProviderResponse) bool {
		return provider.Provider == models.ProviderPassword && provider.Email == originalEmail
	}))

	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, originalEmail, reloaded.Email)
}

func TestUpdateAccount_RefusesEmailNotFromSignInMethod(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	body, err := json.Marshal(map[string]string{"email": "stranger@example.com"})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodPatch, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "Choose an email from a connected sign-in method.")
}

func TestDisconnectAccountProvider_AllowsWhenPasswordExists(t *testing.T) {
	r := support.Setup(t)
	hash, err := crypto.HashPassword("current-pass-123")
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	server, account, token := setupTestServer(r, t)

	req, _ := http.NewRequest(http.MethodDelete, "/account/providers/github", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
	_, err = account.GetAccountProvider(models.ProviderGitHub)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDisconnectAccountProvider_MovesEmailToRemainingProvider(t *testing.T) {
	r := support.Setup(t)
	hash, err := crypto.HashPassword("current-pass-123")
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Create(&models.AccountProvider{
		AccountID:  r.Account.ID,
		Provider:   models.ProviderGoogle,
		ProviderID: "google-ada",
		Email:      "ada-google@example.com",
		Username:   "ada",
		Name:       r.Account.Name,
	}).Error)

	server, account, token := setupTestServer(r, t)
	req, _ := http.NewRequest(http.MethodDelete, "/account/providers/github", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "ada-google@example.com", reloaded.Email)
}

func TestDisconnectAccountProvider_KeepsEmailWhenNextEmailIsTaken(t *testing.T) {
	r := support.Setup(t)
	hash, err := crypto.HashPassword("current-pass-123")
	require.NoError(t, err)
	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	other, err := models.CreateAccount("Other", "ada-google@example.com")
	require.NoError(t, err)
	require.NotEmpty(t, other.ID)
	require.NoError(t, database.Conn().Create(&models.AccountProvider{
		AccountID:  r.Account.ID,
		Provider:   models.ProviderGoogle,
		ProviderID: "google-ada",
		Email:      "ada-google@example.com",
		Username:   "ada",
		Name:       r.Account.Name,
	}).Error)

	originalEmail := r.Account.Email
	server, account, token := setupTestServer(r, t)
	req, _ := http.NewRequest(http.MethodDelete, "/account/providers/github", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
	reloaded, err := models.FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, originalEmail, reloaded.Email)
	_, err = account.GetAccountProvider(models.ProviderGitHub)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDeleteAccount_SoftDeletesCreatedOrgAndFreesEmail(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	server, account, token := setupTestServer(r, t)
	originalEmail := account.Email

	body, err := json.Marshal(map[string]string{"email": originalEmail})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)

	_, err = models.FindAccountByID(account.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindAccountByEmail(originalEmail)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindAccountByProvider(models.ProviderGitHub, "testuser")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	replacement, err := models.CreateAccount("Other", originalEmail)
	require.NoError(t, err)
	assert.Equal(t, originalEmail, replacement.Email)
}

func TestDeleteAccount_LeavesOwnedButUncreatedOrg(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	other, err := models.CreateAccount("Other Owner", "other-owner@example.com")
	require.NoError(t, err)
	require.NoError(t, models.SetOrganizationCreatedByAccount(database.Conn(), r.Organization.ID, other.ID))

	ownerIDs, err := r.AuthService.GetOrgUsersForRole(t.Context(), models.RoleOrgOwner, r.Organization.ID.String())
	require.NoError(t, err)
	require.Contains(t, ownerIDs, r.User.String())

	server, _, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusConflict, res.Code)
	assert.Contains(t, res.Body.String(), "Transfer ownership")

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
}

func TestDeleteAccount_RefusesLastInstallationAdmin(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	server, _, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusConflict, res.Code)
	assert.Contains(t, res.Body.String(), "installation admin")

	_, err = models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
}

func TestDeleteAccount_SkipsDeletedOrganizationMembership(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)
	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	server, _, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
}

func TestDeleteAccount_RequiresEmailConfirmation(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	body, err := json.Marshal(map[string]string{"email": "wrong@example.com"})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestLinkProviderToAccount_RefusesForeignIdentity(t *testing.T) {
	r := support.Setup(t)
	other, err := models.CreateAccount("Other", "other-sso@example.com")
	require.NoError(t, err)

	err = authentication.LinkProviderToAccount(r.Encryptor, other, testGothUser("google", "google-1", "other-sso@example.com"))
	require.NoError(t, err)

	err = authentication.LinkProviderToAccount(r.Encryptor, r.Account, testGothUser("google", "google-1", "other-sso@example.com"))
	assert.ErrorIs(t, err, models.ErrSignInIdentityInUse)
}

func TestLinkProviderToAccount_AttachesUnusedIdentity(t *testing.T) {
	r := support.Setup(t)

	err := authentication.LinkProviderToAccount(r.Encryptor, r.Account, testGothUser("google", "google-2", "ada@example.com"))
	require.NoError(t, err)

	linked, err := models.FindAccountByProvider(models.ProviderGoogle, "google-2")
	require.NoError(t, err)
	assert.Equal(t, r.Account.ID, linked.ID)
}

func testGothUser(provider, providerID, email string) goth.User {
	return goth.User{
		Provider: provider,
		UserID:   providerID,
		Email:    email,
		Name:     "Linked",
		NickName: "linked",
	}
}
