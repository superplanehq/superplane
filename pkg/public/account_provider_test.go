package public

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
)

func TestAccountResponseIncludesProviders(t *testing.T) {
	resources := support.Setup(t)
	signer := jwt.NewSigner("test-secret")
	server, err := NewServer(
		resources.Encryptor,
		resources.Registry,
		signer,
		support.NewOIDCProvider(),
		resources.GitProvider,
		"",
		"",
		"",
		"test",
		"/app/templates",
		resources.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	token, err := authentication.GenerateAccountToken(
		signer,
		resources.Account.ID.String(),
		time.Now(),
		time.Hour,
	)
	require.NoError(t, err)

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/account",
		authCookie: token,
	})
	require.Equal(t, http.StatusOK, response.Code)

	var account AccountResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&account))
	require.Len(t, account.Providers, 1)
	assert.Equal(t, "github", account.Providers[0].Provider)
	assert.Equal(t, "testuser", account.Providers[0].Username)
	assert.Equal(t, "test@example.com", account.Providers[0].DisplayName)
	assert.Equal(t, "test", account.Providers[0].Email)
	assert.Equal(t, "https://github.com/testuser.png", account.Providers[0].AvatarURL)
}

func TestGitHubAccountLinkRoutesRequireAccountAuthentication(t *testing.T) {
	resources := support.Setup(t)
	server, err := NewServer(
		resources.Encryptor,
		resources.Registry,
		jwt.NewSigner("test-secret"),
		support.NewOIDCProvider(),
		resources.GitProvider,
		"",
		"",
		"",
		"test",
		"/app/templates",
		resources.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)

	for _, test := range []struct {
		path   string
		status int
	}{
		{path: "/account/providers/github/connect", status: http.StatusUnauthorized},
		{path: "/auth/github/callback/link", status: http.StatusTemporaryRedirect},
	} {
		response := execRequest(server, requestParams{method: http.MethodGet, path: test.path})
		assert.Equal(t, test.status, response.Code, test.path)
	}
}

func TestGitHubAccountLinkRejectsImpersonation(t *testing.T) {
	account := &models.Account{ID: uuid.New()}
	request := httptest.NewRequest(http.MethodGet, "/account/providers/github/connect", nil)
	ctx := context.WithValue(request.Context(), middleware.AccountContextKey, account)
	ctx = context.WithValue(ctx, middleware.ImpersonationContextKey, &middleware.ImpersonationInfo{Active: true})
	response := httptest.NewRecorder()

	(&Server{}).connectGitHubAccount(response, request.WithContext(ctx))

	assert.Equal(t, http.StatusForbidden, response.Code)
}
