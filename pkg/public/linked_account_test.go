package public

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DisconnectLinkedAccountRemovesBitbucketUUID(t *testing.T) {
	server, registry, token := newLinkedAccountServer(t)
	const providerID = "{11111111-1111-1111-1111-111111111111}"
	saveLinkedAccount(t, registry, models.ProviderBitbucket, providerID, "ada-bb")

	response := deleteLinkedAccount(t, server, token, models.ProviderBitbucket, providerID)

	require.Equal(t, http.StatusNoContent, response.Code)
	assertLinkedAccountAbsent(t, registry, models.ProviderBitbucket, providerID)
}

func Test__DisconnectLinkedAccountRemovesGitHubAccount(t *testing.T) {
	server, registry, token := newLinkedAccountServer(t)
	const providerID = "4242"
	saveLinkedAccount(t, registry, models.ProviderGitHub, providerID, "ada-gh")

	response := deleteLinkedAccount(t, server, token, models.ProviderGitHub, providerID)

	require.Equal(t, http.StatusNoContent, response.Code)
	assertLinkedAccountAbsent(t, registry, models.ProviderGitHub, providerID)
}

func Test__DisconnectLinkedAccountRejectsUnknownProvider(t *testing.T) {
	server, registry, token := newLinkedAccountServer(t)
	const providerID = "77"
	saveLinkedAccount(t, registry, "gitlab", providerID, "ada-gl")

	response := deleteLinkedAccount(t, server, token, "gitlab", providerID)

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "Unknown linked account")
	assertLinkedAccountPresent(t, registry, "gitlab", providerID)
}

func Test__DisconnectLinkedAccountReturnsNotFoundWhenBitbucketIDIsMissing(t *testing.T) {
	server, registry, token := newLinkedAccountServer(t)
	const savedID = "{22222222-2222-2222-2222-222222222222}"
	const missingID = "{33333333-3333-3333-3333-333333333333}"
	saveLinkedAccount(t, registry, models.ProviderBitbucket, savedID, "ada-bb")

	response := deleteLinkedAccount(t, server, token, models.ProviderBitbucket, missingID)

	require.Equal(t, http.StatusNotFound, response.Code)
	assert.Contains(t, response.Body.String(), "Linked account not found")
	assertLinkedAccountPresent(t, registry, models.ProviderBitbucket, savedID)
}

func newLinkedAccountServer(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Helper()
	registry := support.Setup(t)
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		registry.Encryptor,
		registry.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"",
		"",
		"test",
		"/app/templates",
		registry.AuthService,
		false,
	)
	require.NoError(t, err)
	token, err := authentication.GenerateAccountToken(signer, registry.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	return server, registry, token
}

func saveLinkedAccount(t *testing.T, registry *support.ResourceRegistry, provider, providerID, username string) {
	t.Helper()
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(registry.Account.ID, provider, providerID, username, "Ada", ""),
	))
}

func deleteLinkedAccount(t *testing.T, server *Server, token, provider, providerID string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, linkedAccountPath(provider, providerID), nil)
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}

func linkedAccountPath(provider, providerID string) string {
	return "/account/linked-accounts/" + provider + "/" + url.PathEscape(providerID)
}

func assertLinkedAccountAbsent(t *testing.T, registry *support.ResourceRegistry, provider, providerID string) {
	t.Helper()
	var count int64
	err := database.DB(t.Context()).
		Model(&models.AccountLinkedAccount{}).
		Where("account_id = ? AND provider = ? AND provider_id = ?", registry.Account.ID, provider, providerID).
		Count(&count).Error
	require.NoError(t, err)
	assert.Zero(t, count)
}

func assertLinkedAccountPresent(t *testing.T, registry *support.ResourceRegistry, provider, providerID string) {
	t.Helper()
	var count int64
	err := database.DB(t.Context()).
		Model(&models.AccountLinkedAccount{}).
		Where("account_id = ? AND provider = ? AND provider_id = ?", registry.Account.ID, provider, providerID).
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
