package authentication

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestOAuthLinkingSessionAge(t *testing.T) {
	for _, intent := range []string{authLinkIntent, authConnectIntent} {
		for _, session := range []string{"fresh", "over age", "missing ses", "malformed ses"} {
			t.Run(intent+"/"+session, func(t *testing.T) {
				handler, account, user := setupOAuthLinking(t, intent)
				now := time.Now()
				start := now
				if session == "over age" {
					start = now.Add(-AccountSessionMaxAge() - time.Hour)
				}
				token := mintAccountToken(t, handler.jwtSigner, account.ID.String(), now, start, time.Hour)
				if session == "missing ses" || session == "malformed ses" {
					claims := map[string]string{"sub": account.ID.String()}
					if session == "malformed ses" {
						claims[sessionStartClaim] = "not-a-timestamp"
					}
					var err error
					token, err = handler.jwtSigner.GenerateWithClaims(time.Hour, claims)
					require.NoError(t, err)
				}
				mustClaims(t, handler.jwtSigner, token)

				request := oauthLinkingRequest(user.Provider, intent, token)
				recorder := httptest.NewRecorder()
				handler.handleAuth(recorder, request)
				if session == "fresh" {
					assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
					location, err := url.Parse(recorder.Header().Get("Location"))
					require.NoError(t, err)
					assert.NotEmpty(t, location.Query().Get("state"))
				} else {
					assert.Equal(t, http.StatusUnauthorized, recorder.Code)
					assert.Empty(t, recorder.Header().Get("Location"))
					assert.Empty(t, recorder.Result().Cookies())
				}

				// A valid signed state must not bypass callback session validation.
				state := signedOAuthLinkingState(t, handler, account, user.Provider, intent)
				request = oauthLinkingRequest(user.Provider, intent, token)
				request.URL.Path += "/callback"
				query := request.URL.Query()
				query.Set("state", state)
				request.URL.RawQuery = query.Encode()
				recorder = httptest.NewRecorder()
				handler.completeProviderAuth(recorder, request, user)

				assert.Equal(t, http.StatusSeeOther, recorder.Code)
				if session != "fresh" {
					assert.Equal(t, "/login?redirect=%2Fsettings%2Faccount%2Fprofile", recorder.Header().Get("Location"))
				} else {
					assert.NotContains(t, recorder.Header().Get("Location"), "/login")
				}
				assertOAuthLinkingPersistence(t, account, user, intent, session == "fresh")
			})
		}
	}
}

func TestOAuthLinkingCompletedProviderSessionRejectsOverAge(t *testing.T) {
	for _, intent := range []string{authLinkIntent, authConnectIntent} {
		t.Run(intent, func(t *testing.T) {
			handler, account, user := setupOAuthLinking(t, intent)
			now := time.Now()
			token := mintAccountToken(t, handler.jwtSigner, account.ID.String(), now, now.Add(-AccountSessionMaxAge()-time.Hour), time.Hour)
			mustClaims(t, handler.jwtSigner, token)
			recorder := httptest.NewRecorder()

			handler.finishProviderAuth(recorder, oauthLinkingRequest(user.Provider, intent, token), user)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assertOAuthLinkingPersistence(t, account, user, intent, false)
		})
	}
}

func TestOAuthLinkingSessionCrossesMaxAge(t *testing.T) {
	for _, intent := range []string{authLinkIntent, authConnectIntent} {
		t.Run(intent, func(t *testing.T) {
			handler, account, user := setupOAuthLinking(t, intent)
			t.Setenv("ACCOUNT_SESSION_MAX_AGE", "3s")
			ResetAccountSessionTTLForTests()
			t.Cleanup(ResetAccountSessionTTLForTests)
			now := time.Now()
			token := mintAccountToken(t, handler.jwtSigner, account.ID.String(), now, now, time.Hour)
			request := oauthLinkingRequest(user.Provider, intent, token)
			recorder := httptest.NewRecorder()
			handler.handleAuth(recorder, request)
			require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
			location, err := url.Parse(recorder.Header().Get("Location"))
			require.NoError(t, err)
			state := location.Query().Get("state")
			require.NotEmpty(t, state)

			claims := mustClaims(t, handler.jwtSigner, token)
			require.Eventually(t, func() bool {
				return !IsAccountSessionWithinMaxAge(claims)
			}, 5*time.Second, 20*time.Millisecond)
			mustClaims(t, handler.jwtSigner, token)
			request = oauthLinkingRequest(user.Provider, intent, token)
			request.URL.Path += "/callback"
			query := request.URL.Query()
			query.Set("state", state)
			request.URL.RawQuery = query.Encode()
			recorder = httptest.NewRecorder()

			handler.completeProviderAuth(recorder, request, user)

			assert.Equal(t, http.StatusSeeOther, recorder.Code)
			assert.Equal(t, "/login?redirect=%2Fsettings%2Faccount%2Fprofile", recorder.Header().Get("Location"))
			assertOAuthLinkingPersistence(t, account, user, intent, false)
		})
	}
}

func setupOAuthLinking(t *testing.T, intent string) (*Handler, *models.Account, goth.User) {
	t.Helper()
	handler, registry := setupAuthHandler(t, false)
	provider := models.ProviderGoogle
	if intent == authConnectIntent {
		provider = models.ProviderBitbucket
	}
	goth.ClearProviders()
	t.Cleanup(goth.ClearProviders)
	handler.InitializeProviders(map[string]ProviderConfig{
		provider: {Key: "dummy-client", Secret: "dummy-secret", CallbackURL: "https://app.example/auth/" + provider + "/callback"},
	})
	require.False(t, registry.Account.IsBlocked())
	require.Nil(t, registry.Account.PasswordChangedAt)
	return handler, registry.Account, goth.User{
		Provider: provider, UserID: uuid.NewString(), NickName: "dummy-user", Name: "Dummy User", Email: "dummy@example.com",
	}
}

func oauthLinkingRequest(provider, intent, token string) *http.Request {
	request := mux.SetURLVars(httptest.NewRequest(http.MethodGet,
		"/auth/"+provider+"?intent="+intent+"&redirect=/settings/account/profile", nil), map[string]string{"provider": provider})
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	return request
}

func signedOAuthLinkingState(t *testing.T, handler *Handler, account *models.Account, provider, intent string) string {
	t.Helper()
	var state string
	var err error
	if intent == authConnectIntent {
		state, err = handler.signConnectState(account.ID.String(), provider, "/settings/account/profile")
	} else {
		state, err = handler.signLinkState(account.ID.String(), provider, "/settings/account/profile")
	}
	require.NoError(t, err)
	return state
}

func assertOAuthLinkingPersistence(t *testing.T, account *models.Account, user goth.User, intent string, accepted bool) {
	t.Helper()
	var credentials, connections int64
	tx := database.DB(t.Context())
	require.NoError(t, tx.Model(&models.AccountProvider{}).
		Where("account_id = ? AND provider = ? AND provider_id = ?", account.ID, user.Provider, user.UserID).Count(&credentials).Error)
	require.NoError(t, tx.Model(&models.AccountLinkedAccount{}).
		Where("account_id = ? AND provider = ? AND provider_id = ?", account.ID, user.Provider, user.UserID).Count(&connections).Error)
	var wantCredentials, wantConnections int64
	if accepted && intent == authLinkIntent {
		wantCredentials = 1
	}
	if accepted && intent == authConnectIntent {
		wantConnections = 1
	}
	assert.Equal(t, wantCredentials, credentials, "sign-in identity persistence")
	assert.Equal(t, wantConnections, connections, "external connection persistence")
}
