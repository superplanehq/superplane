package public

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/mcpserver"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/supportsession"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

const testCurrentPassword = "current-pass-123"
const testNewPassword = "new-secure-pass-456"

func passwordMCPCode(t *testing.T, server *Server, cookie *http.Cookie, org, factory, client string) string {
	t.Helper()
	query := url.Values{"client_id": {client}, "redirect_uri": {mcpserver.CursorRedirectURIs[0]}, "response_type": {"code"}, "code_challenge": {mcp.S256Challenge(mcpTestPKCEVerifier)}, "code_challenge_method": {"S256"}}
	request := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query.Encode(), nil)
	request.Host = "localhost:8000"
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	form := url.Values{"consent": {mcpConsentToken(t, response.Body.String())}, "organization_id": {org}, "workspaces_for": {org}, "factory_id": {factory}}
	request = httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	request.Host = "localhost:8000"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookie)
	response = httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	require.Equal(t, http.StatusFound, response.Code, response.Body.String())
	location, err := url.Parse(response.Header().Get("Location"))
	require.NoError(t, err)
	return location.Query().Get("code")
}

func passwordMCPRefresh(server *Server, refresh, client string) *httptest.ResponseRecorder {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}, "client_id": {client}, "resource": {"http://localhost:8000/mcp"}}
	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Host = "localhost:8000"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	return response
}

func passwordMCPTokens(t *testing.T, response *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var body struct {
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.NotEmpty(t, body.Access)
	require.NotEmpty(t, body.Refresh)
	return body.Access, body.Refresh
}

func TestChangePassword_RevokeMCPLifecycle(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, "Password fixture", "", "PWD")
	require.NoError(t, err)
	task, err := factory.CreateWorkOrder(database.DB(t.Context()), "Harmless password fixture", "", &r.User, nil, nil)
	require.NoError(t, err)
	cookie := &http.Cookie{Name: "account_token", Value: token}
	code := func() string {
		return passwordMCPCode(t, server, cookie, r.Organization.ID.String(), factory.ID.String(), mcpserver.LocalClientID)
	}
	exchange := func(code string) *httptest.ResponseRecorder {
		return exchangeMCPCode(server, code, mcpserver.LocalClientID, mcpserver.CursorRedirectURIs[0])
	}
	access, refresh := passwordMCPTokens(t, exchange(code()))
	readTask := func(access string, status int) {
		t.Helper()
		payload := mustMarshal(t, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "get_task", "arguments": map[string]any{"task": task.ID.String()}}})
		request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
		request.Host = "localhost:8000"
		request.Header.Set("Authorization", "Bearer "+access)
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, request)
		require.Equal(t, status, response.Code, response.Body.String())
		if status == http.StatusOK {
			var body struct {
				Error  any `json:"error"`
				Result struct {
					IsError bool `json:"isError"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"result"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Nil(t, body.Error)
			require.False(t, body.Result.IsError)
			require.NotEmpty(t, body.Result.Content)
			require.Contains(t, body.Result.Content[0].Text, task.ID.String())
		}
	}
	readTask(access, http.StatusOK)
	oldClaims, err := mcpserver.ParseAccessToken(server.jwt, access, "http://localhost:8000/mcp")
	require.NoError(t, err)
	legacyClaims := *oldClaims
	legacyClaims.PasswordVersion = ""
	legacy, err := mcpserver.MintAccessToken(server.jwt, legacyClaims, time.Hour)
	require.NoError(t, err)
	readTask(legacy, http.StatusOK)
	pending := code()
	delayedCode := code()
	db := database.DB(t.Context())
	otherOrg, err := models.CreateOrganization("password-other", "Password other")
	require.NoError(t, err)
	otherUser, err := models.CreateUser(otherOrg.ID, r.Account.ID, r.Account.Email, r.Account.Name)
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, otherOrg.ID, "Other fixture", "", "OTH")
	require.NoError(t, err)
	otherRequest := &mcpserver.AuthorizeRequest{ClientID: "dummy-other-client", RedirectURI: mcpserver.CursorRedirectURIs[0], Resource: "http://localhost:8000/mcp", CodeChallenge: mcp.S256Challenge(mcpTestPKCEVerifier), CodeChallengeMethod: "S256"}
	otherCode, err := mcpserver.CreateAuthorizationCode(db, otherRequest, otherUser.ID, otherOrg.ID, []uuid.UUID{otherFactory.ID})
	require.NoError(t, err)
	_, otherRefresh := passwordMCPTokens(t, exchangeMCPCode(server, otherCode, otherRequest.ClientID, otherRequest.RedirectURI))
	_, err = mcpserver.CreateAuthorizationCode(db, otherRequest, otherUser.ID, otherOrg.ID, []uuid.UUID{otherFactory.ID})
	require.NoError(t, err)
	require.NoError(t, db.Delete(otherUser).Error)
	unrelatedAccount, err := models.CreateAccount("Unrelated", "unrelated-password@example.com")
	require.NoError(t, err)
	unrelatedUser, err := models.CreateUser(r.Organization.ID, unrelatedAccount.ID, unrelatedAccount.Email, unrelatedAccount.Name)
	require.NoError(t, err)
	unrelatedCode, err := mcpserver.CreateAuthorizationCode(db, otherRequest, unrelatedUser.ID, r.Organization.ID, []uuid.UUID{factory.ID})
	require.NoError(t, err)
	_, unrelatedRefresh := passwordMCPTokens(t, exchangeMCPCode(server, unrelatedCode, otherRequest.ClientID, otherRequest.RedirectURI))
	var delayedIssue *mcpserver.TokenIssue
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		var oauthErr *mcpserver.OAuthError
		delayedIssue, _, oauthErr = mcpserver.ExchangeAuthorizationCode(tx, url.Values{"grant_type": {"authorization_code"}, "code": {delayedCode}, "code_verifier": {mcpTestPKCEVerifier}, "client_id": {mcpserver.LocalClientID}, "redirect_uri": {mcpserver.CursorRedirectURIs[0]}}, "http://localhost:8000/mcp")
		if oauthErr != nil {
			return oauthErr
		}
		return nil
	}))
	failed := sendChangePasswordRequest(server, mustMarshal(t, changePasswordRequest{CurrentPassword: "wrong", NewPassword: testNewPassword}), cookie)
	require.Equal(t, http.StatusBadRequest, failed.Code)
	readTask(access, http.StatusOK)
	access, refresh = passwordMCPTokens(t, passwordMCPRefresh(server, refresh, mcpserver.LocalClientID))
	rotated := sendChangePasswordRequest(server, mustMarshal(t, changePasswordRequest{CurrentPassword: testCurrentPassword, NewPassword: testNewPassword}), cookie)
	require.Equal(t, http.StatusNoContent, rotated.Code)
	for _, model := range []any{&models.MCPOAuthCode{}, &models.MCPOAuthRefreshToken{}} {
		var remaining int64
		require.NoError(t, db.Model(model).Where("user_id IN ?", []uuid.UUID{r.User, otherUser.ID}).Count(&remaining).Error)
		require.Zero(t, remaining)
	}
	require.Equal(t, http.StatusBadRequest, passwordMCPRefresh(server, otherRefresh, otherRequest.ClientID).Code)
	passwordMCPTokens(t, passwordMCPRefresh(server, unrelatedRefresh, otherRequest.ClientID))
	for _, renewed := range rotated.Result().Cookies() {
		if renewed.Name == "account_token" {
			cookie = renewed
		}
	}
	readTask(access, http.StatusUnauthorized)
	readTask(legacy, http.StatusUnauthorized)
	for _, rejected := range []*httptest.ResponseRecorder{passwordMCPRefresh(server, refresh, mcpserver.LocalClientID), exchange(pending)} {
		require.Equal(t, http.StatusBadRequest, rejected.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rejected.Body.Bytes(), &body))
		require.Equal(t, "invalid_grant", body["error"])
	}
	freshAccess, freshRefresh := passwordMCPTokens(t, exchange(code()))
	readTask(freshAccess, http.StatusOK)
	passwordMCPTokens(t, passwordMCPRefresh(server, freshRefresh, mcpserver.LocalClientID))
	readTask(access, http.StatusUnauthorized)
	delayedClaims := *oldClaims
	delayedClaims.PasswordVersion = delayedIssue.PasswordVersion
	delayed, err := mcpserver.MintAccessToken(server.jwt, delayedClaims, time.Hour)
	require.NoError(t, err)
	readTask(delayed, http.StatusUnauthorized)
	// Keep a matching fresh grant and put rotation in this JWT's issuance second.
	freshClaims, err := mcpserver.ParseAccessToken(server.jwt, freshAccess, "http://localhost:8000/mcp")
	require.NoError(t, err)
	sameSecond, err := mcpserver.MintAccessToken(server.jwt, *freshClaims, time.Hour)
	require.NoError(t, err)
	issued, err := server.jwt.ValidateAndGetClaims(sameSecond)
	require.NoError(t, err)
	account, err := models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
	rotationTime := time.Unix(int64(issued["iat"].(float64)), 0).Add(999999 * time.Microsecond)
	require.NoError(t, account.MarkPasswordChangedInTransaction(db, rotationTime))
	readTask(sameSecond, http.StatusUnauthorized)
}

func TestChangePassword_RejectsConcurrentMCPConsent(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, "Consent fixture", "", "CNS")
	require.NoError(t, err)
	consent, err := mcpserver.MintConsentToken(server.jwt, mcpserver.ConsentClaims{ClientID: mcpserver.LocalClientID, RedirectURI: mcpserver.CursorRedirectURIs[0], Resource: "http://localhost:8000/mcp", CodeChallenge: mcp.S256Challenge(mcpTestPKCEVerifier), CodeChallengeMethod: "S256", ResponseType: "code"})
	require.NoError(t, err)
	type consentRaceKey struct{}
	ctx, cancel := context.WithTimeout(context.WithValue(t.Context(), consentRaceKey{}, true), 15*time.Second)
	defer cancel()
	paused, resume := make(chan struct{}), make(chan struct{})
	defer close(resume)
	callback := "password_consent_" + uuid.NewString()
	queries := database.Conn().Callback().Query()
	require.NoError(t, queries.Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Context.Value(consentRaceKey{}) != true || tx.Statement.Table != "accounts" {
			return
		}
		if _, locking := tx.Statement.Clauses["FOR"]; !locking {
			return
		}
		close(paused)
		select {
		case <-resume:
		case <-ctx.Done():
		}
	}))
	defer func() { require.NoError(t, queries.Remove(callback)) }()
	form := url.Values{"consent": {consent}, "organization_id": {r.Organization.ID.String()}, "workspaces_for": {r.Organization.ID.String()}, "factory_id": {factory.ID.String()}}
	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode())).WithContext(ctx)
	request.Host = "localhost:8000"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cookie := &http.Cookie{Name: "account_token", Value: token}
	request.AddCookie(cookie)
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, request)
		done <- response
	}()
	select {
	case <-paused:
	case <-ctx.Done():
		t.Fatal("consent did not reach the account lock")
	}
	rotated := sendChangePasswordRequest(server, mustMarshal(t, changePasswordRequest{CurrentPassword: testCurrentPassword, NewPassword: testNewPassword}), cookie)
	require.Equal(t, http.StatusNoContent, rotated.Code)
	resume <- struct{}{}
	select {
	case response := <-done:
		require.Equal(t, http.StatusUnauthorized, response.Code)
	case <-ctx.Done():
		t.Fatal("consent did not finish")
	}
	var codes int64
	require.NoError(t, database.DB(t.Context()).Model(&models.MCPOAuthCode{}).Where("user_id = ?", r.User).Count(&codes).Error)
	require.Zero(t, codes)
}

func TestChangePassword_MCPExchangeRace(t *testing.T) {
	for _, grant := range []string{"code", "refresh"} {
		for _, rotationFirst := range []bool{true, false} {
			name := grant + "/exchange-first"
			if rotationFirst {
				name = grant + "/rotation-first"
			}
			t.Run(name, func(t *testing.T) {
				server, r, token := setupChangePasswordTestServer(t)
				require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
				require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
				factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, "Race fixture", "", "RAC")
				require.NoError(t, err)
				cookie := &http.Cookie{Name: "account_token", Value: token}
				code := passwordMCPCode(t, server, cookie, r.Organization.ID.String(), factory.ID.String(), mcpserver.LocalClientID)
				exchange := func() *httptest.ResponseRecorder {
					return exchangeMCPCode(server, code, mcpserver.LocalClientID, mcpserver.CursorRedirectURIs[0])
				}
				if grant == "refresh" {
					_, refresh := passwordMCPTokens(t, exchange())
					exchange = func() *httptest.ResponseRecorder { return passwordMCPRefresh(server, refresh, mcpserver.LocalClientID) }
				}
				race := &acceptanceRace{t: t}
				release := race.holdRow(&models.Account{}, r.Account.ID)
				rotation := func() *httptest.ResponseRecorder {
					return sendChangePasswordRequest(server, mustMarshal(t, changePasswordRequest{CurrentPassword: testCurrentPassword, NewPassword: testNewPassword}), cookie)
				}
				start := func(operation func() *httptest.ResponseRecorder) <-chan *httptest.ResponseRecorder {
					done := make(chan *httptest.ResponseRecorder, 1)
					go func() { done <- operation() }()
					return done
				}
				var exchanged, rotated <-chan *httptest.ResponseRecorder
				if rotationFirst {
					rotated = start(rotation)
				} else {
					exchanged = start(exchange)
				}
				race.waitForLockWaiters(1)
				if rotationFirst {
					exchanged = start(exchange)
				} else {
					rotated = start(rotation)
				}
				race.waitForLockWaiters(2)
				release()
				finish := func(done <-chan *httptest.ResponseRecorder) *httptest.ResponseRecorder {
					select {
					case response := <-done:
						return response
					case <-time.After(10 * time.Second):
						t.Fatal("MCP exchange and password rotation did not finish")
						return nil
					}
				}
				require.Equal(t, http.StatusNoContent, finish(rotated).Code)
				response := finish(exchanged)
				if rotationFirst {
					require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
					require.Contains(t, response.Body.String(), `"error":"invalid_grant"`)
					return
				}
				access, refresh := passwordMCPTokens(t, response)
				require.Equal(t, http.StatusBadRequest, passwordMCPRefresh(server, refresh, mcpserver.LocalClientID).Code)
				request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
				request.Host = "localhost:8000"
				request.Header.Set("Authorization", "Bearer "+access)
				denied := httptest.NewRecorder()
				server.Router.ServeHTTP(denied, request)
				require.Equal(t, http.StatusUnauthorized, denied.Code)
			})
		}
	}
}

// setupChangePasswordTestServer constructs a server with password login
// enabled and seeds a password auth row for the test account so the
// change-password handler can exercise the success path.
func setupChangePasswordTestServer(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "yes")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	hash, err := crypto.HashPassword(testCurrentPassword)
	require.NoError(t, err)

	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func setupChangePasswordTestServerNoPassword(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "yes")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func setupChangePasswordTestServerLoginDisabled(t *testing.T) (*Server, *support.ResourceRegistry, string) {
	t.Setenv("ENABLE_PASSWORD_LOGIN", "no")

	r := support.Setup(t)

	signer := jwt.NewSigner("test-client-secret")
	server, err := NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")

	hash, err := crypto.HashPassword(testCurrentPassword)
	require.NoError(t, err)

	_, err = models.CreateAccountPasswordAuth(r.Account.ID, hash)
	require.NoError(t, err)

	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)

	return server, r, token
}

func sendChangePasswordRequest(server *Server, body []byte, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPost, "/account/password", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	res := httptest.NewRecorder()
	server.Router.ServeHTTP(res, req)
	return res
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestChangePassword_Unauthenticated(t *testing.T) {
	server, _, _ := setupChangePasswordTestServer(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestChangePassword_PasswordLoginDisabled(t *testing.T) {
	server, _, token := setupChangePasswordTestServerLoginDisabled(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_AccountWithoutPasswordAuth(t *testing.T) {
	server, _, token := setupChangePasswordTestServerNoPassword(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_RejectsSupportSession(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)

	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))

	target, err := models.CreateAccount("Target User", "target-support-session@example.com")
	require.NoError(t, err)

	signer := jwt.NewSigner("test-client-secret")
	sessionToken, err := supportsession.GenerateToken(signer, r.Account.ID.String(), target.ID.String())
	require.NoError(t, err)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body,
		&http.Cookie{Name: "account_token", Value: token},
		&http.Cookie{Name: supportsession.CookieName, Value: sessionToken},
	)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChangePassword_ValidationErrors(t *testing.T) {
	server, _, token := setupChangePasswordTestServer(t)

	cases := []struct {
		name string
		body any
	}{
		{
			name: "missing current password",
			body: map[string]string{"newPassword": testNewPassword},
		},
		{
			name: "missing new password",
			body: map[string]string{"currentPassword": testCurrentPassword},
		},
		{
			name: "new password too short",
			body: map[string]string{"currentPassword": testCurrentPassword, "newPassword": "short"},
		},
		{
			name: "new password equals current",
			body: map[string]string{"currentPassword": testCurrentPassword, "newPassword": testCurrentPassword},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := sendChangePasswordRequest(server, mustMarshal(t, c.body),
				&http.Cookie{Name: "account_token", Value: token})
			assert.Equal(t, http.StatusBadRequest, res.Code)
		})
	}
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	server, _, token := setupChangePasswordTestServer(t)

	body := mustMarshal(t, map[string]string{
		"currentPassword": "wrong-password",
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: token})
	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestChangePassword_Success(t *testing.T) {
	server, r, token := setupChangePasswordTestServer(t)

	// Seed an existing API token on the user so we can verify it gets cleared.
	originalTokenHash := crypto.HashToken("some-existing-api-token")
	require.NoError(t, r.UserModel.UpdateTokenHash(originalTokenHash))

	// Seed a named personal API token so we can verify it gets revoked too.
	personalToken := models.NewUserAPIToken(r.User, "CI token", crypto.HashToken("some-personal-token"))
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), personalToken))

	// Drop a stale support session cookie on the request to verify it is cleared.
	staleSupportSession := &http.Cookie{Name: supportsession.CookieName, Value: "stale-token-value"}

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})

	res := sendChangePasswordRequest(server, body,
		&http.Cookie{Name: "account_token", Value: token},
		staleSupportSession,
	)
	require.Equal(t, http.StatusNoContent, res.Code)

	// Password hash is rotated.
	passwordAuth, err := models.FindAccountPasswordAuthByAccountID(r.Account.ID)
	require.NoError(t, err)
	assert.True(t, crypto.VerifyPassword(passwordAuth.PasswordHash, testNewPassword))
	assert.False(t, crypto.VerifyPassword(passwordAuth.PasswordHash, testCurrentPassword))

	// password_changed_at is set.
	updatedAccount, err := models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
	require.NotNil(t, updatedAccount.PasswordChangedAt)
	assert.WithinDuration(t, time.Now(), *updatedAccount.PasswordChangedAt, 5*time.Second)

	// Every user under the account had token_hash cleared.
	var users []models.User
	require.NoError(t, database.Conn().Where("account_id = ?", r.Account.ID).Find(&users).Error)
	require.NotEmpty(t, users)
	for _, u := range users {
		assert.Empty(t, u.TokenHash, "token_hash should have been cleared for user %s", u.ID)
	}

	// Every named personal API token was revoked.
	tokens, err := models.ListUserAPITokens(database.Conn(), r.User)
	require.NoError(t, err)
	assert.Empty(t, tokens, "personal API tokens should have been revoked on password change")

	// Cookies were set: a refreshed account_token AND a cleared support_session_token.
	cookies := res.Result().Cookies()
	var newAuthCookie, clearedSessionCookie *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case "account_token":
			newAuthCookie = c
		case supportsession.CookieName:
			clearedSessionCookie = c
		}
	}

	require.NotNil(t, newAuthCookie, "a fresh account_token cookie should be issued")
	assert.NotEqual(t, token, newAuthCookie.Value, "the reissued account_token must not equal the old one")
	assert.True(t, newAuthCookie.HttpOnly)
	assert.NotZero(t, newAuthCookie.MaxAge)

	require.NotNil(t, clearedSessionCookie, "the support session cookie should be cleared")
	assert.Equal(t, "", clearedSessionCookie.Value)
	assert.Equal(t, -1, clearedSessionCookie.MaxAge)
}

func TestGetAccount_HasPassword(t *testing.T) {
	t.Run("returns false when account has no password auth", func(t *testing.T) {
		os.Unsetenv("ENABLE_PASSWORD_LOGIN")
		server, _, token := setupTestServer(support.Setup(t), t)

		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var resp AccountResponse
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
		assert.False(t, resp.HasPassword)
	})

	t.Run("returns true when account has password auth", func(t *testing.T) {
		server, r, token := setupChangePasswordTestServer(t)

		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)

		require.Equal(t, http.StatusOK, res.Code)
		var resp AccountResponse
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &resp))
		assert.True(t, resp.HasPassword)
		assert.Equal(t, r.Account.ID.String(), resp.ID)
	})
}

// TestChangePassword_InvalidatesOldCookie verifies the freshness check
// in the middleware: an account_token issued before password_changed_at
// is rejected, while a fresh one issued after is accepted.
func TestChangePassword_InvalidatesOldCookie(t *testing.T) {
	server, r, oldToken := setupChangePasswordTestServer(t)

	// Confirm the cookie works before any password change.
	check := func(token string) int {
		req, _ := http.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
		res := httptest.NewRecorder()
		server.Router.ServeHTTP(res, req)
		return res.Code
	}

	require.Equal(t, http.StatusOK, check(oldToken))

	// Wait a second so the new cookie's iat is strictly newer than the
	// old cookie's, since JWT iat resolution is whole seconds.
	time.Sleep(1100 * time.Millisecond)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})
	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: oldToken})
	require.Equal(t, http.StatusNoContent, res.Code)

	// The old cookie is now stale and rejected.
	assert.Equal(t, http.StatusUnauthorized, check(oldToken))

	// A freshly minted cookie with iat after password_changed_at is accepted.
	signer := jwt.NewSigner("test-client-secret")
	freshToken, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, check(freshToken))
}

// TestChangePassword_ReissuedCookieValid verifies the cookie returned
// by the change-password handler itself is treated as fresh.
func TestChangePassword_ReissuedCookieValid(t *testing.T) {
	server, _, oldToken := setupChangePasswordTestServer(t)

	time.Sleep(1100 * time.Millisecond)

	body := mustMarshal(t, map[string]string{
		"currentPassword": testCurrentPassword,
		"newPassword":     testNewPassword,
	})
	res := sendChangePasswordRequest(server, body, &http.Cookie{Name: "account_token", Value: oldToken})
	require.Equal(t, http.StatusNoContent, res.Code)

	var newAuthCookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == "account_token" {
			newAuthCookie = c
			break
		}
	}
	require.NotNil(t, newAuthCookie)

	req, _ := http.NewRequest(http.MethodGet, "/account", nil)
	req.AddCookie(newAuthCookie)
	check := httptest.NewRecorder()
	server.Router.ServeHTTP(check, req)
	assert.Equal(t, http.StatusOK, check.Code)
}

// TestSetAccountCookie_Helper sanity-checks the new cookie helper so any
// future drift in cookie attributes is caught explicitly.
func TestSetAccountCookie_Helper(t *testing.T) {
	for _, test := range []struct {
		name   string
		url    string
		header string
		secure bool
	}{
		{name: "local HTTP", url: "http://superplane.test/account", secure: false},
		{name: "direct HTTPS", url: "https://superplane.test/account", secure: true},
		{name: "forwarded HTTPS", url: "http://superplane.test/account", header: "https", secure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, test.url, nil)
			r.Header.Set("X-Forwarded-Proto", test.header)

			authentication.SetAccountCookie(w, r, "abc", time.Hour)

			cookies := w.Result().Cookies()
			require.Len(t, cookies, 1)
			c := cookies[0]
			assert.Equal(t, "account_token", c.Name)
			assert.Equal(t, "abc", c.Value)
			assert.Equal(t, "/", c.Path)
			assert.True(t, c.HttpOnly)
			assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
			assert.Equal(t, int(time.Hour.Seconds()), c.MaxAge)
			assert.Equal(t, test.secure, c.Secure)
		})
	}
}
