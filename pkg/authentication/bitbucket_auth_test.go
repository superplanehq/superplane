package authentication

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestUseRealProviderAuthInDevelopment_UsesBitbucketConnect(t *testing.T) {
	connect := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket?intent=connect", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	assert.True(t, useRealProviderAuthInDevelopment(connect))

	signIn := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	assert.False(t, useRealProviderAuthInDevelopment(signIn))
}

func TestHandler_handleAuthConfig_OmitsBitbucket(t *testing.T) {
	t.Cleanup(goth.ClearProviders)
	goth.ClearProviders()

	handler := &Handler{}
	handler.InitializeProviders(map[string]ProviderConfig{
		models.ProviderGitHub: {
			Key:         "github-id",
			Secret:      "github-secret",
			CallbackURL: "https://app.example/auth/github/callback",
		},
		models.ProviderBitbucket: {
			Key:         "bitbucket-id",
			Secret:      "bitbucket-secret",
			CallbackURL: "https://app.example/auth/bitbucket/callback",
		},
	})

	recorder := httptest.NewRecorder()
	handler.handleAuthConfig(recorder, httptest.NewRequest(http.MethodGet, "/auth/config", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Providers        []string `json:"providers"`
		ConnectProviders []string `json:"connectProviders"`
	}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	assert.Contains(t, response.Providers, models.ProviderGitHub)
	assert.NotContains(t, response.Providers, models.ProviderBitbucket)
	assert.Equal(t, []string{models.ProviderBitbucket}, response.ConnectProviders)
}

func TestHandler_handleAuthConfig_OmitsUnconfiguredBitbucketLink(t *testing.T) {
	t.Cleanup(goth.ClearProviders)
	goth.ClearProviders()

	handler := &Handler{}
	handler.InitializeProviders(map[string]ProviderConfig{
		models.ProviderGitHub: {
			Key:         "github-id",
			Secret:      "github-secret",
			CallbackURL: "https://app.example/auth/github/callback",
		},
		models.ProviderBitbucket: {
			CallbackURL: "https://app.example/auth/bitbucket/callback",
		},
	})

	recorder := httptest.NewRecorder()
	handler.handleAuthConfig(recorder, httptest.NewRequest(http.MethodGet, "/auth/config", nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Providers        []string `json:"providers"`
		ConnectProviders []string `json:"connectProviders"`
	}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	assert.NotContains(t, response.Providers, models.ProviderBitbucket)
	assert.Empty(t, response.ConnectProviders)
}

func TestHandler_handleAuth_RejectsBitbucketSignIn(t *testing.T) {
	handler := &Handler{}
	for _, target := range []string{"/auth/bitbucket", "/auth/bitbucket?intent=link"} {
		request := mux.SetURLVars(
			httptest.NewRequest(http.MethodGet, target, nil),
			map[string]string{"provider": models.ProviderBitbucket},
		)
		recorder := httptest.NewRecorder()

		handler.handleAuth(recorder, request)

		assert.Equal(t, http.StatusForbidden, recorder.Code, target)
		assert.Empty(t, cookieValue(recorder, "account_token"), target)
		assert.NotContains(t, recorder.Header().Get("Location"), "bitbucket.org", target)
	}
}

func TestHandler_handleDevelopmentAuth_RejectsBitbucketSignIn(t *testing.T) {
	handler := NewHandler(jwt.NewSigner("test-secret"), nil, nil, "development", "/templates", false, false, false)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	for _, target := range []string{"/auth/bitbucket", "/auth/bitbucket?intent=link"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

		assert.Equal(t, http.StatusForbidden, recorder.Code, target)
		assert.Empty(t, cookieValue(recorder, "account_token"), target)
		assert.NotContains(t, recorder.Header().Get("Location"), "bitbucket.org", target)
	}
}

func TestHandler_handleAuth_BitbucketConnect(t *testing.T) {
	t.Cleanup(goth.ClearProviders)

	t.Run("returns an error when the consumer is not configured", func(t *testing.T) {
		goth.ClearProviders()
		handler, registry := setupAuthHandler(t, false)
		request := bitbucketConnectRequest(t, handler, registry.Account.ID.String())
		recorder := httptest.NewRecorder()

		handler.handleAuth(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "Bitbucket is not configured")
		assert.Empty(t, cookieValue(recorder, "account_token"))
		assert.Empty(t, recorder.Header().Get("Location"))
	})

	t.Run("redirects a signed-in account to Bitbucket", func(t *testing.T) {
		goth.ClearProviders()
		handler, registry := setupAuthHandler(t, false)
		handler.InitializeProviders(map[string]ProviderConfig{
			models.ProviderBitbucket: {
				Key:         "consumer-key",
				Secret:      "consumer-secret",
				CallbackURL: "https://app.example/auth/bitbucket/callback",
			},
		})
		request := bitbucketConnectRequest(t, handler, registry.Account.ID.String())
		recorder := httptest.NewRecorder()

		handler.handleAuth(recorder, request)

		require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
		location, err := url.Parse(recorder.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "https", location.Scheme)
		assert.Equal(t, "bitbucket.org", location.Host)
		assert.Equal(t, "/site/oauth2/authorize", location.Path)
		assert.Equal(t, bitbucketAccountScope, location.Query().Get("scope"))
		assert.NotEmpty(t, location.Query().Get("state"))
		assert.Empty(t, cookieValue(recorder, "account_token"))
	})
}

func TestHandler_handleAuthCallback_RejectsBitbucketWithoutConnectState(t *testing.T) {
	handler, registry := setupAuthHandler(t, false)
	original := gothic.CompleteUserAuth
	t.Cleanup(func() { gothic.CompleteUserAuth = original })

	otherAccount, err := models.CreateAccount("Other", "bitbucket-other@example.com")
	require.NoError(t, err)
	otherState, err := handler.signConnectState(otherAccount.ID.String(), models.ProviderBitbucket, "/settings")
	require.NoError(t, err)
	validState, err := handler.signConnectState(registry.Account.ID.String(), models.ProviderBitbucket, "/settings")
	require.NoError(t, err)

	cases := []struct {
		name    string
		target  string
		session bool
	}{
		{name: "missing state", target: "/auth/bitbucket/callback"},
		{name: "sign-in state", target: "/auth/bitbucket/callback?state=" + url.QueryEscape("/settings")},
		{name: "link state", target: "/auth/bitbucket/callback?state=link:not-a-token"},
		{name: "connect state for another account", target: "/auth/bitbucket/callback?state=" + url.QueryEscape(otherState), session: true},
		{name: "connect state without a session", target: "/auth/bitbucket/callback?state=" + url.QueryEscape(validState)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			gothic.CompleteUserAuth = func(http.ResponseWriter, *http.Request) (goth.User, error) {
				called = true
				return goth.User{}, nil
			}
			request := mux.SetURLVars(
				httptest.NewRequest(http.MethodGet, tc.target, nil),
				map[string]string{"provider": models.ProviderBitbucket},
			)
			if tc.session {
				request.AddCookie(accountSessionCookie(t, handler, registry.Account.ID.String()))
			}
			recorder := httptest.NewRecorder()

			handler.handleAuthCallback(recorder, request)

			assert.Equal(t, http.StatusForbidden, recorder.Code)
			assert.False(t, called)
			assert.Empty(t, cookieValue(recorder, "account_token"))
		})
	}
}

func TestHandler_handleAuthCallback_StoresBitbucketLink(t *testing.T) {
	handler, registry := setupAuthHandler(t, false)
	original := gothic.CompleteUserAuth
	t.Cleanup(func() { gothic.CompleteUserAuth = original })

	const rawID = "{504C3B1A-1001-4000-8000-000000000001}"
	expectedID := uuid.MustParse(rawID).String()
	gothic.CompleteUserAuth = func(http.ResponseWriter, *http.Request) (goth.User, error) {
		return goth.User{
			UserID:       rawID,
			NickName:     "ada",
			Name:         "Ada Lovelace",
			AvatarURL:    "https://bitbucket.org/account/ada/avatar/32/",
			Email:        "bitbucket-link-should-not-create@example.com",
			Provider:     models.ProviderBitbucket,
			AccessToken:  "account-token",
			RefreshToken: "refresh-token",
		}, nil
	}

	state, err := handler.signConnectState(registry.Account.ID.String(), models.ProviderBitbucket, "/settings/account/profile")
	require.NoError(t, err)
	request := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket/callback?state="+url.QueryEscape(state), nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	request.AddCookie(accountSessionCookie(t, handler, registry.Account.ID.String()))
	recorder := httptest.NewRecorder()

	handler.handleAuthCallback(recorder, request)

	require.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "/settings/account/profile?linked_account=linked&provider=bitbucket", recorder.Header().Get("Location"))
	assert.Empty(t, cookieValue(recorder, "account_token"))

	linked, err := models.FindAccountLinkedAccount(database.Conn(), registry.Account.ID, models.ProviderBitbucket)
	require.NoError(t, err)
	assert.Equal(t, expectedID, linked.ProviderID)
	assert.Equal(t, "ada", linked.Username)
	assert.Equal(t, "Ada Lovelace", linked.Name)
	assert.Equal(t, "https://bitbucket.org/account/ada/avatar/32/", linked.AvatarURL)

	_, err = registry.Account.FindAccountProviderByID(models.ProviderBitbucket, expectedID)
	assert.Error(t, err)
	_, err = models.FindAccountByEmail("bitbucket-link-should-not-create@example.com")
	assert.Error(t, err)
}

func TestHandler_completeProviderAuth_RejectsBitbucketSignIn(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)
	user := goth.User{
		UserID:       "{504c3b1a-1001-4000-8000-000000000009}",
		Email:        "bitbucket-signin@example.com",
		NickName:     "ada",
		Provider:     models.ProviderBitbucket,
		AccessToken:  "account-token",
		RefreshToken: "refresh-token",
	}
	request := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket/callback", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	recorder := httptest.NewRecorder()

	handler.completeProviderAuth(recorder, request, user)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Empty(t, cookieValue(recorder, "account_token"))
	_, err := models.FindAccountByEmail(user.Email)
	assert.Error(t, err)
	var providers int64
	require.NoError(t, database.Conn().Model(&models.AccountProvider{}).
		Where("provider = ?", models.ProviderBitbucket).
		Count(&providers).Error)
	assert.Zero(t, providers)
}

func bitbucketConnectRequest(t *testing.T, handler *Handler, accountID string) *http.Request {
	t.Helper()
	request := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket?intent=connect&redirect=/settings/account/profile", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	request.AddCookie(accountSessionCookie(t, handler, accountID))
	return request
}

func accountSessionCookie(t *testing.T, handler *Handler, accountID string) *http.Cookie {
	t.Helper()
	token, err := handler.jwtSigner.GenerateWithClaims(time.Hour, map[string]string{
		"sub":             accountID,
		sessionStartClaim: strconv.FormatInt(time.Now().Unix(), 10),
	})
	require.NoError(t, err)
	return &http.Cookie{Name: "account_token", Value: token}
}
