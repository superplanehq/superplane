package authentication

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAccountLinkState(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)

	state, err := handler.newAccountLinkState("account-1", "/settings/personal?tab=accounts")
	require.NoError(t, err)

	claims, err := handler.parseAccountLinkState(state)
	require.NoError(t, err)
	assert.Equal(t, "account-1", claims.AccountID)
	assert.Equal(t, "/settings/personal?tab=accounts", claims.Redirect)
	assert.NotEmpty(t, claims.Nonce)

	otherHandler, _ := setupAuthHandler(t, false)
	otherHandler.jwtSigner.Secret = "another-secret"
	_, err = otherHandler.parseAccountLinkState(state)
	assert.Error(t, err)
}

func TestAccountLinkStateRejectsExpiredState(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)
	claims := accountLinkStateClaims{
		AccountID: "account-1",
		Redirect:  "/settings/personal",
		Nonce:     "nonce",
		Purpose:   accountLinkStatePurpose,
		StandardClaims: jwtLib.StandardClaims{
			ExpiresAt: time.Now().Add(-time.Minute).Unix(),
		},
	}
	token := jwtLib.NewWithClaims(jwtLib.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(handler.jwtSigner.Secret))
	require.NoError(t, err)

	_, err = handler.parseAccountLinkState(state)
	assert.Error(t, err)
}

func TestAccountLinkCallbackRejectsAnotherAccount(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)
	state, err := handler.newAccountLinkState("initiating-account", "/settings/personal")
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "/auth/github/callback/link?state="+state, nil)
	response := httptest.NewRecorder()
	handler.CompleteGitHubAccountLink(response, request, &models.Account{})

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Empty(t, response.Header().Values("Set-Cookie"))
}

func TestAccountLinkCallbackReturnsDeniedWithoutChangingSession(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)
	account := &models.Account{ID: uuid.New()}
	state, err := handler.newAccountLinkState(account.ID.String(), "/settings/personal")
	require.NoError(t, err)

	request := httptest.NewRequest(
		http.MethodGet,
		"/auth/github/callback/link?error=access_denied&state="+state,
		nil,
	)
	response := httptest.NewRecorder()
	handler.CompleteGitHubAccountLink(response, request, account)

	assert.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, "/settings/personal?provider=github&provider_link=denied", response.Header().Get("Location"))
	assert.Empty(t, response.Header().Values("Set-Cookie"))
}

func TestSafeAccountLinkRedirect(t *testing.T) {
	assert.Equal(t, "/settings/personal?tab=accounts", safeAccountLinkRedirect("/settings/personal?tab=accounts"))
	assert.Equal(t, "/", safeAccountLinkRedirect("https://attacker.example/settings"))
	assert.Equal(t, "/", safeAccountLinkRedirect("//attacker.example/settings"))
	assert.Equal(t, "/", safeAccountLinkRedirect(`/\attacker.example/settings`))
	assert.Equal(t, "/", safeAccountLinkRedirect("settings/personal"))
}

func TestLinkGitHubAccount(t *testing.T) {
	t.Run("links and reconnects the same identity without changing the account", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		account, err := models.CreateAccount("Account User", "account@example.com")
		require.NoError(t, err)

		user := githubAccountLinkUser{
			ID:          "github-1",
			Username:    "octocat",
			Name:        "Octo Cat",
			Email:       "octocat@github.example",
			AvatarURL:   "https://avatars.example/octocat",
			AccessToken: "token-1",
		}
		require.NoError(t, handler.linkGitHubAccount(t.Context(), account, user))

		user.Name = "Updated Name"
		user.AccessToken = "token-2"
		require.NoError(t, handler.linkGitHubAccount(t.Context(), account, user))

		providers, err := models.ListAccountProviders(database.DB(t.Context()), account.ID)
		require.NoError(t, err)
		require.Len(t, providers, 1)
		assert.Equal(t, "Updated Name", providers[0].Name)

		storedAccount, err := models.FindAccountByID(account.ID.String())
		require.NoError(t, err)
		assert.Equal(t, "account@example.com", storedAccount.Email)
	})

	t.Run("rejects an identity linked to another account", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		first, err := models.CreateAccount("First", "first@example.com")
		require.NoError(t, err)
		second, err := models.CreateAccount("Second", "second@example.com")
		require.NoError(t, err)

		user := githubAccountLinkUser{ID: "github-shared", Email: "shared@github.example", AccessToken: "token"}
		require.NoError(t, handler.linkGitHubAccount(t.Context(), first, user))

		err = handler.linkGitHubAccount(t.Context(), second, user)
		assert.True(t, errors.Is(err, models.ErrProviderLinkedToAnotherAccount))
	})

	t.Run("rejects a different identity for the same account", func(t *testing.T) {
		handler, _ := setupAuthHandler(t, false)
		account, err := models.CreateAccount("Account", "account@example.com")
		require.NoError(t, err)

		require.NoError(t, handler.linkGitHubAccount(t.Context(), account, githubAccountLinkUser{
			ID: "github-first", Email: "first@github.example", AccessToken: "token",
		}))

		err = handler.linkGitHubAccount(t.Context(), account, githubAccountLinkUser{
			ID: "github-second", Email: "second@github.example", AccessToken: "token",
		})
		assert.True(t, errors.Is(err, models.ErrAccountProviderConflict))
	})
}
