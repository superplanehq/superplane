package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/markbates/goth"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/polarfake"
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

func TestGetAccount_ListsOrganizationsPendingDeletion(t *testing.T) {
	r := support.Setup(t)
	server, _, token := setupTestServer(r, t)

	req, _ := http.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var resp AccountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	require.Len(t, resp.OrganizationsPendingDeletion, 1)
	assert.Equal(t, r.Organization.ID.String(), resp.OrganizationsPendingDeletion[0].ID)
	assert.Equal(t, r.Organization.Name, resp.OrganizationsPendingDeletion[0].Name)

	otherAccount, err := models.CreateAccount("Other Owner", "other-owner-get@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(database.Conn(), r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(database.Conn(), otherUser.ID, true))

	req, _ = http.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res = httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
	assert.Empty(t, resp.OrganizationsPendingDeletion)
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

	_, err = models.FindAccountByProvider(database.Conn(), models.ProviderGitHub, "testuser")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	replacement, err := models.CreateAccount("Other", originalEmail)
	require.NoError(t, err)
	assert.Equal(t, originalEmail, replacement.Email)
}

func TestDeleteAccount_LeavesCreatedOrgWhenOtherOwnerRemains(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	otherAccount, err := models.CreateAccount("Other Owner", "other-created-owner@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(database.Conn(), r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(database.Conn(), otherUser.ID, true))
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
	_, err = models.ApplyPolarSubscription(database.Conn(), r.Organization.ID, models.PolarSubscriptionApply{
		ID: "sub_other_owner", ProductID: "prod_business", Status: models.PolarSubscriptionStatusActive,
	}, nil)
	require.NoError(t, err)

	server, account, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)

	_, err = models.FindAccountByID(account.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)

	_, err = models.FindActiveHumanUserByAccountAndOrganization(database.Conn(), r.Organization.ID, account.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	ownerIDs, err := models.ListOrganizationOwnerIDs(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.NotContains(t, ownerIDs, r.User.String())
	assert.Contains(t, ownerIDs, otherUser.ID.String())
}

func TestDeleteAccount_ResolvesSeatsOverCapacityInRemainingOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	require.NoError(t, db.Model(r.Account).Update("installation_admin", false).Error)
	otherAccount, err := models.CreateAccount("Other Owner", "seats-other-owner@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(db, r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(db, otherUser.ID, true))

	product := "prod_team_" + uuid.NewString()
	team := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "public", PerSeat: true, SeatLimit: new(50), PriceCents: 2000, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, models.SaveBillingPlan(db, &team))
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
		"plan_id": team.ID, "plan": team.Key, "plan_source": models.BillingPlanSourcePolar,
		"polar_subscription_id": "sub_" + uuid.NewString(), "polar_subscription_status": models.PolarSubscriptionStatusActive,
		"purchased_seats": 1, "over_capacity_since": time.Now().Add(-time.Hour),
	}).Error)

	previousHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	hook := &logtest.Hook{}
	logrus.AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(previousHooks) })

	server, account, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)

	var resolved []*logrus.Entry
	for _, entry := range hook.AllEntries() {
		if entry.Message == logging.SeatsOverCapacityResolvedMessage {
			resolved = append(resolved, entry)
		}
	}
	require.Len(t, resolved, 1)
	assert.Equal(t, logging.ComponentBillingSeats, resolved[0].Data["component"])
	assert.Equal(t, "account_deletion", resolved[0].Data["cause"])
	assert.Equal(t, r.Organization.ID.String(), resolved[0].Data["organization_id"])
	assert.Equal(t, 1, resolved[0].Data["occupied"])
	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, plan.OverCapacitySince)
}

func TestDeleteAccount_RefusesLiveSubscriptionAndPreservesAccount(t *testing.T) {
	for _, status := range []string{models.PolarSubscriptionStatusActive, models.PolarSubscriptionStatusPaused} {
		t.Run(status, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, db.Model(r.Account).Update("installation_admin", false).Error)
			second := support.CreateOrganization(t, r, r.User)
			// Place the subscribed organization last so rejection must also roll
			// back deletion of an earlier organization in the same account.
			protected := r.Organization
			if second.ID.String() > protected.ID.String() {
				protected = second
			}
			require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
			_, err := models.ApplyPolarSubscription(db, protected.ID, models.PolarSubscriptionApply{
				ID: "sub_account_delete", ProductID: "prod_business", Status: status,
			}, nil)
			require.NoError(t, err)
			useAccountDeletionPolar(t, func(w http.ResponseWriter, orgID string) {
				require.Equal(t, protected.ID.String(), orgID)
				writePolarSubscriptions(t, w, map[string]any{"id": "sub_account_delete", "status": status, "external_customer_id": orgID})
			})
			server, account, token := setupTestServer(r, t)
			beforeAccount, err := models.FindAccountByID(account.ID.String())
			require.NoError(t, err)
			beforeUsers, err := models.ListActiveHumanUsersForAccount(db, account.ID)
			require.NoError(t, err)
			beforeProviders, err := account.GetAccountProviders(db)
			require.NoError(t, err)
			beforePlan, err := models.FindOrganizationBillingPlan(db, protected.ID)
			require.NoError(t, err)
			body, err := json.Marshal(map[string]string{"email": account.Email})
			require.NoError(t, err)
			deleteAccount := func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
				res := httptest.NewRecorder()
				server.Router.ServeHTTP(res, req)
				return res
			}

			res := deleteAccount()
			require.Equal(t, http.StatusConflict, res.Code)
			assert.Contains(t, res.Body.String(), "Cancel the subscription in Billing")
			afterAccount, err := models.FindAccountByID(account.ID.String())
			require.NoError(t, err)
			assert.Equal(t, beforeAccount, afterAccount)
			afterUsers, err := models.ListActiveHumanUsersForAccount(db, account.ID)
			require.NoError(t, err)
			assert.ElementsMatch(t, beforeUsers, afterUsers)
			afterProviders, err := account.GetAccountProviders(db)
			require.NoError(t, err)
			assert.ElementsMatch(t, beforeProviders, afterProviders)
			afterPlan, err := models.FindOrganizationBillingPlan(db, protected.ID)
			require.NoError(t, err)
			assert.Equal(t, beforePlan, afterPlan)
			for _, organization := range []*models.Organization{r.Organization, second} {
				stored, err := models.FindOrganizationByID(organization.ID.String())
				require.NoError(t, err)
				assert.False(t, stored.DeletedAt.Valid)
			}

			_, err = models.ApplyPolarSubscription(db, protected.ID, models.PolarSubscriptionApply{
				ID: "sub_account_delete", ProductID: "prod_business", Status: models.PolarSubscriptionStatusCanceled,
			}, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, deleteAccount().Code)
			_, err = models.FindAccountByID(account.ID.String())
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
			for _, organization := range []*models.Organization{r.Organization, second} {
				_, err := models.FindOrganizationByID(organization.ID.String())
				require.ErrorIs(t, err, gorm.ErrRecordNotFound)
			}
		})
	}
}

func TestDeleteAccount_AsksPolarAboutElapsedScheduledCancellation(t *testing.T) {
	setup := func(t *testing.T) (*support.ResourceRegistry, func() *httptest.ResponseRecorder) {
		t.Helper()
		r := support.Setup(t)
		db := database.DB(t.Context())
		require.NoError(t, db.Model(r.Account).Update("installation_admin", false).Error)
		require.NoError(t, models.BindLegacyBillingProduct(db, "prod_business"))
		end := time.Now().Add(-time.Hour)
		start := end.AddDate(0, -1, 0)
		_, err := models.ApplyPolarSubscription(db, r.Organization.ID, models.PolarSubscriptionApply{
			ID: "sub_elapsed", ProductID: "prod_business", Status: models.PolarSubscriptionStatusActive,
			PeriodStart: &start, PeriodEnd: &end, CancelAtPeriodEnd: true,
		}, nil)
		require.NoError(t, err)
		server, account, token := setupTestServer(r, t)
		body, err := json.Marshal(map[string]string{"email": account.Email})
		require.NoError(t, err)
		return r, func() *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
			req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
			res := httptest.NewRecorder()
			server.Router.ServeHTTP(res, req)
			return res
		}
	}
	requireAccountAndOrganizationKept := func(t *testing.T, r *support.ResourceRegistry) {
		t.Helper()
		_, err := models.FindAccountByID(r.Account.ID.String())
		require.NoError(t, err)
		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
	}

	t.Run("provider resumed the subscription", func(t *testing.T) {
		r, deleteAccount := setup(t)
		useAccountDeletionPolar(t, func(w http.ResponseWriter, orgID string) {
			writePolarSubscriptions(t, w, map[string]any{"id": "sub_elapsed", "status": "active", "external_customer_id": orgID})
		})

		res := deleteAccount()

		require.Equal(t, http.StatusConflict, res.Code)
		assert.Contains(t, res.Body.String(), "Cancel the subscription in Billing")
		requireAccountAndOrganizationKept(t, r)
	})

	t.Run("provider failure", func(t *testing.T) {
		r, deleteAccount := setup(t)
		useAccountDeletionPolar(t, func(w http.ResponseWriter, _ string) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		res := deleteAccount()

		require.Equal(t, http.StatusServiceUnavailable, res.Code)
		assert.Contains(t, res.Body.String(), models.ErrOrganizationSubscriptionUnverified.Error())
		requireAccountAndOrganizationKept(t, r)
	})

	t.Run("provider confirmed termination", func(t *testing.T) {
		r, deleteAccount := setup(t)
		useAccountDeletionPolar(t, func(w http.ResponseWriter, orgID string) {
			writePolarSubscriptions(t, w, map[string]any{"id": "sub_elapsed", "status": "canceled", "external_customer_id": orgID})
		})

		res := deleteAccount()

		require.Equal(t, http.StatusNoContent, res.Code)
		_, err := models.FindAccountByID(r.Account.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func useAccountDeletionPolar(t *testing.T, respond func(w http.ResponseWriter, orgID string)) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subscriptions/", r.URL.Path)
		respond(w, r.URL.Query().Get("external_customer_id"))
	}))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
}

func writePolarSubscriptions(t *testing.T, w http.ResponseWriter, items ...map[string]any) {
	t.Helper()
	assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]any{"max_page": 1}}))
}

func TestDeleteAccount_LeavesOwnedButUncreatedOrg(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	other, err := models.CreateAccount("Other Owner", "other-owner@example.com")
	require.NoError(t, err)
	require.NoError(t, models.SetOrganizationCreatedByAccount(database.Conn(), r.Organization.ID, other.ID))

	ownerIDs, err := models.ListOrganizationOwnerIDs(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	require.Contains(t, ownerIDs, r.User.String())

	server, account, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
	assert.NotContains(t, res.Body.String(), "Transfer ownership")

	_, err = models.FindAccountByID(account.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
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

func TestDeleteAccount_RemovesOrganizationRoles(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	server, account, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)

	ownerIDs, err := models.ListOrganizationOwnerIDs(database.Conn(), r.Organization.ID)
	require.NoError(t, err)
	assert.NotContains(t, ownerIDs, r.User.String())
}

func TestDeleteAccount_RefusesLastLivingOwnerAfterDeletedOwner(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.Conn().Model(r.Account).Update("installation_admin", false).Error)

	creator, err := models.CreateAccount("Creator", "org-creator@example.com")
	require.NoError(t, err)
	require.NoError(t, models.SetOrganizationCreatedByAccount(database.Conn(), r.Organization.ID, creator.ID))

	otherAccount, err := models.CreateAccount("Other Owner", "other-living-owner@example.com")
	require.NoError(t, err)

	otherUser, err := models.CreateUserInTransaction(database.Conn(), r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(database.Conn(), otherUser.ID, true))

	firstServer, _, firstToken := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: firstToken})
	res := httptest.NewRecorder()
	firstServer.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)

	otherToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), otherAccount.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	body, err = json.Marshal(map[string]string{"email": otherAccount.Email})
	require.NoError(t, err)
	req, _ = http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: otherToken})
	res = httptest.NewRecorder()
	firstServer.Router.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)
	assert.NotContains(t, res.Body.String(), "Transfer ownership")

	_, err = models.FindAccountByID(otherAccount.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDeleteAccount_RefusesWhenOnlyOtherAdminIsBlocked(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	blocked, err := models.CreateAccount("Blocked Admin", "blocked-admin@example.com")
	require.NoError(t, err)
	require.NoError(t, models.PromoteToInstallationAdmin(blocked.ID.String()))
	require.NoError(t, blocked.Block(database.Conn(), time.Now()))

	server, _, token := setupTestServer(r, t)
	body, err := json.Marshal(map[string]string{"email": r.Account.Email})
	require.NoError(t, err)
	req, _ := http.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)

	require.Equal(t, http.StatusConflict, res.Code)
	assert.Contains(t, res.Body.String(), "installation admin")
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

	err = authentication.LinkProviderToAccount(database.Conn(), other, testGothUser("google", "google-1", "other-sso@example.com"))
	require.NoError(t, err)

	err = authentication.LinkProviderToAccount(database.Conn(), r.Account, testGothUser("google", "google-1", "other-sso@example.com"))
	assert.ErrorIs(t, err, models.ErrSignInIdentityInUse)
}

func TestLinkProviderToAccount_AttachesUnusedIdentity(t *testing.T) {
	r := support.Setup(t)

	err := authentication.LinkProviderToAccount(database.Conn(), r.Account, testGothUser("google", "google-2", "ada@example.com"))
	require.NoError(t, err)

	linked, err := models.FindAccountByProvider(database.Conn(), models.ProviderGoogle, "google-2")
	require.NoError(t, err)
	assert.Equal(t, r.Account.ID, linked.ID)
}

func TestLinkProviderToAccount_RefusesSharedGitHubIdentity(t *testing.T) {
	support.Setup(t)
	first, err := models.CreateAccount("First GitHub", "first-github-sso@example.com")
	require.NoError(t, err)
	second, err := models.CreateAccount("Second GitHub", "second-github-sso@example.com")
	require.NoError(t, err)

	err = authentication.LinkProviderToAccount(database.Conn(), first, testGothUser(models.ProviderGitHub, "github-shared", first.Email))
	require.NoError(t, err)

	err = authentication.LinkProviderToAccount(database.Conn(), second, testGothUser(models.ProviderGitHub, "github-shared", second.Email))
	assert.ErrorIs(t, err, models.ErrSignInIdentityInUse)
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

// accountDeletionRenewalFixture is a seat-plan organization that keeps
// running after the test account is deleted, because another owner remains.
type accountDeletionRenewalFixture struct {
	r        *support.ResourceRegistry
	provider *polarfake.Server
	sub      polarfake.Subscription
}

func newAccountDeletionRenewalFixture(t *testing.T) *accountDeletionRenewalFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "yes")
	db := database.Conn()
	require.NoError(t, db.Model(r.Account).Update("installation_admin", false).Error)
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	addOrganizationMember(t, r.Organization.ID, true)
	addOrganizationMember(t, r.Organization.ID, false)

	provider := polarfake.NewServer(t)
	t.Cleanup(seatrenewal.Wait)
	// Five paid seats, three members, and a renewal in one hour.
	sub := provider.AddSeatPlan(db, r.Organization.ID, 5, time.Now().Add(time.Hour))
	return &accountDeletionRenewalFixture{r: r, provider: provider, sub: sub}
}

func addOrganizationMember(t *testing.T, organizationID uuid.UUID, owner bool) {
	t.Helper()
	account, err := models.CreateAccount("Member", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	user, err := models.CreateUserInTransaction(database.Conn(), organizationID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(database.Conn(), user.ID, owner))
}

func (f *accountDeletionRenewalFixture) deleteAccount(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	server, account, token := setupTestServer(f.r, t)
	body, err := json.Marshal(map[string]string{"email": account.Email})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	seatrenewal.Wait()
	return res
}

func (f *accountDeletionRenewalFixture) plan(t *testing.T) *models.OrganizationBillingPlan {
	t.Helper()
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	return plan
}

func requestsFor(requests []polarfake.Request, subscriptionID string) []polarfake.Request {
	var matching []polarfake.Request
	for _, request := range requests {
		if request.SubscriptionID == subscriptionID {
			matching = append(matching, request)
		}
	}
	return matching
}

func TestDeleteAccount_SchedulesRenewalForSurvivingOrganization(t *testing.T) {
	f := newAccountDeletionRenewalFixture(t)

	res := f.deleteAccount(t)

	require.Equal(t, http.StatusNoContent, res.Code)
	assert.Equal(t, []polarfake.Request{{
		Method: http.MethodPatch, SubscriptionID: f.sub.ID,
		Body: map[string]any{"seats": float64(2), "proration_behavior": "next_period"},
	}}, f.provider.Patches())
	assert.Equal(t, new(2), f.provider.Subscription(f.sub.ID).PendingSeats)
	assert.Equal(t, 5, f.provider.Subscription(f.sub.ID).Seats, "the current period keeps every paid seat")
	assert.Equal(t, new(2), f.plan(t).RenewalSeatsScheduled)
}

func TestDeleteAccount_DoesNotScheduleRenewalForDeletedOrganization(t *testing.T) {
	f := newAccountDeletionRenewalFixture(t)
	db := database.Conn()
	deleted, err := models.CreateOrganizationInTransaction(db, support.RandomName("org"), "")
	require.NoError(t, err)
	require.NoError(t, models.SetOrganizationCreatedByAccount(db, deleted.ID, f.r.Account.ID))
	user, err := models.CreateUserInTransaction(db, deleted.ID, f.r.Account.ID, f.r.Account.Email, f.r.Account.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(db, user.ID, true))
	// The subscription of the deleted organization ended, so deletion needs
	// no Polar check. A schedule for it would still read it from Polar.
	ended := f.provider.AddSeatPlan(db, deleted.ID, 2, time.Now().Add(time.Hour))
	f.provider.Update(ended.ID, func(sub *polarfake.Subscription) { sub.Status = models.PolarSubscriptionStatusCanceled })
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", deleted.ID).
		Update("polar_subscription_status", models.PolarSubscriptionStatusCanceled).Error)

	res := f.deleteAccount(t)

	require.Equal(t, http.StatusNoContent, res.Code)
	_, err = models.FindOrganizationByID(deleted.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Empty(t, requestsFor(f.provider.Requests(), ended.ID))
	assert.Len(t, requestsFor(f.provider.Patches(), f.sub.ID), 1, "the surviving organization still gets its schedule")
}

func TestDeleteAccount_SucceedsWhenPolarRejectsTheRenewalSchedule(t *testing.T) {
	f := newAccountDeletionRenewalFixture(t)
	f.provider.FailNext(1, http.StatusUnprocessableEntity, "PolarRequestValidationError")

	res := f.deleteAccount(t)

	require.Equal(t, http.StatusNoContent, res.Code)
	_, err := models.FindAccountByID(f.r.Account.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindActiveHumanUserByAccountAndOrganization(database.Conn(), f.r.Organization.ID, f.r.Account.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Empty(t, f.provider.Patches())
	assert.Nil(t, f.provider.Subscription(f.sub.ID).PendingSeats)
	assert.Nil(t, f.plan(t).RenewalSeatsScheduled, "the seat renewal sync worker schedules it later")
}
