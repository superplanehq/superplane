package authentication

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/markbates/goth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestCompleteAccountConnectionSavesBitbucketUUID(t *testing.T) {
	handler, registry := setupAuthHandler(t, false)
	const accountUUID = "{11111111-1111-1111-1111-111111111111}"
	recorder := httptest.NewRecorder()
	request := accountSessionRequest(t, handler, registry.Account.ID.String())
	state := connectStateFor(t, handler, registry.Account.ID.String(), models.ProviderBitbucket)

	handler.completeAccountConnection(recorder, request, goth.User{
		Provider: models.ProviderBitbucket,
		UserID:   accountUUID,
		NickName: "ada-bb",
		Name:     "Ada",
	}, state)

	require.Equal(t, http.StatusSeeOther, recorder.Code)
	linked, err := models.FindAccountLinkedAccount(database.Conn(), registry.Account.ID, models.ProviderBitbucket)
	require.NoError(t, err)
	assert.Equal(t, accountUUID, linked.ProviderID)
	assert.Equal(t, "ada-bb", linked.Username)
}

func TestCompleteAccountConnectionRejectsInvalidGitHubID(t *testing.T) {
	handler, registry := setupAuthHandler(t, false)
	recorder := httptest.NewRecorder()
	request := accountSessionRequest(t, handler, registry.Account.ID.String())
	state := connectStateFor(t, handler, registry.Account.ID.String(), models.ProviderGitHub)

	handler.completeAccountConnection(recorder, request, goth.User{
		Provider: models.ProviderGitHub,
		UserID:   "not-a-number",
		NickName: "octocat",
	}, state)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "GitHub returned an invalid user ID")
	_, err := models.FindAccountLinkedAccount(database.Conn(), registry.Account.ID, models.ProviderGitHub)
	require.Error(t, err)
}

func TestBitbucketSignInWithoutConnectIntentIsRejected(t *testing.T) {
	handler, _ := setupAuthHandler(t, false)
	request := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	recorder := httptest.NewRecorder()

	handler.handleAuth(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Bitbucket does not support sign-in")

	handler.isDev = true
	devRequest := mux.SetURLVars(
		httptest.NewRequest(http.MethodGet, "/auth/bitbucket", nil),
		map[string]string{"provider": models.ProviderBitbucket},
	)
	devRecorder := httptest.NewRecorder()
	handler.handleDevelopmentAuth(devRecorder, devRequest)
	require.Equal(t, http.StatusBadRequest, devRecorder.Code)
	assert.Contains(t, devRecorder.Body.String(), "Bitbucket does not support sign-in")
}

func accountSessionRequest(t *testing.T, handler *Handler, accountID string) *http.Request {
	t.Helper()
	token, err := handler.jwtSigner.GenerateWithClaims(time.Hour, map[string]string{
		"sub":             accountID,
		sessionStartClaim: strconv.FormatInt(time.Now().Unix(), 10),
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/auth/bitbucket/callback", nil)
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	return request
}

func connectStateFor(t *testing.T, handler *Handler, accountID, provider string) *connectState {
	t.Helper()
	signed, err := handler.signConnectState(accountID, provider, "/settings/account/profile")
	require.NoError(t, err)
	state, err := handler.parseConnectState(signed)
	require.NoError(t, err)
	return state
}
