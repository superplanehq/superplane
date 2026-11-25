package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type fakeOIDC struct {
	issuer          string
	key             *rsa.PrivateKey
	nonce           string
	audience        string
	expired         bool
	subject         string
	email           string
	omitEmail       bool
	userinfoSubject string
	extraClaims     map[string]any
	discoveryStatus int
	discoveryBody   string
}

func newFakeOIDC(t *testing.T) (*fakeOIDC, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	fake := &fakeOIDC{key: key, audience: "client-1", subject: "user-1", email: "owner@example.com"}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	fake.issuer = server.URL
	t.Cleanup(server.Close)
	return fake, server
}

func (f *fakeOIDC) serve(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		if f.discoveryStatus != 0 {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(f.discoveryStatus)
			_, _ = w.Write([]byte(f.discoveryBody))
			return
		}
		writeJSON(w, map[string]any{
			"issuer":                                f.issuer,
			"authorization_endpoint":                f.issuer + "/authorize",
			"token_endpoint":                        f.issuer + "/token",
			"jwks_uri":                              f.issuer + "/jwks",
			"userinfo_endpoint":                     f.issuer + "/userinfo",
			"end_session_endpoint":                  f.issuer + "/logout",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/jwks":
		writeJSON(w, map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "test",
			"n": base64.RawURLEncoding.EncodeToString(f.key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.key.E)).Bytes()),
		}}})
	case "/token":
		token := jwtLib.NewWithClaims(jwtLib.SigningMethodRS256, f.claims())
		token.Header["kid"] = "test"
		signed, err := token.SignedString(f.key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"access_token": "access", "token_type": "Bearer", "id_token": signed})
	case "/userinfo":
		subject := f.userinfoSubject
		if subject == "" {
			subject = f.subject
		}
		writeJSON(w, map[string]any{"sub": subject, "email": f.email, "name": "Owner"})
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeOIDC) claims() jwtLib.MapClaims {
	now := time.Now()
	exp := now.Add(time.Hour)
	if f.expired {
		now = time.Now().Add(-2 * time.Hour)
		exp = time.Now().Add(-time.Minute)
	}
	claims := jwtLib.MapClaims{
		"iss": f.issuer, "sub": f.subject, "aud": f.audience,
		"exp": exp.Unix(), "iat": now.Unix(), "nonce": f.nonce,
		"name": "Owner", "sid": "sid-1",
	}
	if !f.omitEmail {
		claims["email"] = f.email
	}
	for key, value := range f.extraClaims {
		claims[key] = value
	}
	return claims
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func TestOIDCSignIn(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	input := connectionInput(true)
	input.Issuer = fake.issuer
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		response := f.callback(t, conn, fake, callbackOptions{})
		require.Equal(t, http.StatusFound, response.Code)
		require.Contains(t, response.Header().Get("Location"), f.r.Organization.Slug)
		require.NotEmpty(t, response.Result().Cookies())
	})

	t.Run("wrong state", func(t *testing.T) {
		response := f.callback(t, conn, fake, callbackOptions{state: "wrong-state"})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
	})

	t.Run("wrong nonce", func(t *testing.T) {
		fake.nonce = "wrong-nonce"
		response := f.callback(t, conn, fake, callbackOptions{keepNonce: true})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
	})

	t.Run("wrong audience", func(t *testing.T) {
		fake.audience = "other-client"
		response := f.callback(t, conn, fake, callbackOptions{})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
		fake.audience = "client-1"
	})

	t.Run("expired token", func(t *testing.T) {
		fake.expired = true
		response := f.callback(t, conn, fake, callbackOptions{})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
		fake.expired = false
	})

	t.Run("missing cookie", func(t *testing.T) {
		response := f.callback(t, conn, fake, callbackOptions{dropCookie: true})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
	})

	t.Run("cookie of another connection", func(t *testing.T) {
		response := f.callback(t, conn, fake, callbackOptions{otherConnection: true})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
	})

	t.Run("refuses a callback after the connection is disabled", func(t *testing.T) {
		start := httptest.NewRequest(http.MethodGet, "/auth/sso/"+conn.ID.String()+"/start", nil)
		started := httptest.NewRecorder()
		f.service.AuthHandler().ServeHTTP(started, start)
		location, err := url.Parse(started.Header().Get("Location"))
		require.NoError(t, err)
		fake.nonce = location.Query().Get("nonce")
		conn.Status = models.SSOStatusDisabled
		require.NoError(t, conn.Save(database.Conn()))
		callback := httptest.NewRequest(http.MethodGet, "/auth/sso/"+conn.ID.String()+"/callback?code=code-1&state="+url.QueryEscape(location.Query().Get("state")), nil)
		for _, cookie := range started.Result().Cookies() {
			callback.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		f.service.AuthHandler().ServeHTTP(response, callback)
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_not_available")
	})
}

func TestAllowedStartQueryDropsUnknownKeys(t *testing.T) {
	query := url.Values{}
	query.Set("login_hint", "ada@example.com")
	query.Set("redirect", "/home")
	query.Set("iss", "https://idp.example.com")
	query.Set("test", "1")
	query.Set("next", "https://evil.example")
	encoded := allowedStartQuery(query).Encode()
	require.Contains(t, encoded, "login_hint=ada%40example.com")
	require.Contains(t, encoded, "redirect=%2Fhome")
	require.Contains(t, encoded, "iss=")
	require.NotContains(t, encoded, "test")
	require.NotContains(t, encoded, "evil")
}

func TestOIDCUserInfoSubject(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	input := connectionInput(true)
	input.Issuer = fake.issuer
	conn := f.put(input)
	f.markTested(conn)
	_, err := f.service.ActivateConnection(f.ctx(), f.r.Organization.ID)
	require.NoError(t, err)
	conn, err = models.FindSSOConnectionByOrganization(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	fake.omitEmail = true

	t.Run("rejects a userinfo subject that differs from the ID token", func(t *testing.T) {
		fake.userinfoSubject = "other-user"
		response := f.callback(t, conn, fake, callbackOptions{})
		require.Contains(t, response.Header().Get("Location"), "auth_error=sso_failed")
	})

	t.Run("uses the userinfo email when the ID token has no email", func(t *testing.T) {
		fake.userinfoSubject = ""
		response := f.callback(t, conn, fake, callbackOptions{})
		require.Equal(t, http.StatusFound, response.Code)
		require.Contains(t, response.Header().Get("Location"), f.r.Organization.Slug)
		saved, err := models.FindSSOIdentityBySubject(database.Conn(), conn.ID, fake.subject)
		require.NoError(t, err)
		require.Equal(t, "owner@example.com", saved.Email)
	})
}

func TestOIDCTestSignIn(t *testing.T) {
	f := newFlow(t)
	f.setOwnerEmail("owner@example.com")
	f.verifyExample()
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	fake, _ := newFakeOIDC(t)
	input := connectionInput(true)
	input.Issuer = fake.issuer
	conn := f.put(input)

	settings := "/" + f.r.Organization.Slug + "/workspaces/line/settings/organization/sso"

	t.Run("refuses a test without a session", func(t *testing.T) {
		response := f.testCallback(t, conn, fake, nil, settings)
		location := response.Header().Get("Location")
		require.Contains(t, location, "auth_error=sso_failed")
		require.Contains(t, location, "redirect="+url.QueryEscape(settings))
		require.NotContains(t, location, "sso_test=")
	})

	t.Run("refuses a member without permission to update SSO", func(t *testing.T) {
		account, err := models.CreateAccount("Operator", "operator@example.com")
		require.NoError(t, err)
		user, err := models.CreateUser(f.r.Organization.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)
		require.NoError(t, f.r.AuthService.AssignRole(user.ID.String(), models.RoleOrgOperator, f.r.Organization.ID.String(), models.DomainTypeOrganization))
		response := f.testCallback(t, conn, fake, account, settings)
		location := response.Header().Get("Location")
		require.Contains(t, location, settings)
		require.Contains(t, location, "sso_test=failed")
		require.Contains(t, location, "reason=test_permission")
		require.NotContains(t, location, "auth_error")
		saved, err := models.FindSSOConnectionByID(database.Conn(), conn.ID)
		require.NoError(t, err)
		require.NotContains(t, string(saved.LastTestResult), "cannot change")
	})

	t.Run("stores the claims and returns to the requested page", func(t *testing.T) {
		response := f.testCallback(t, conn, fake, f.r.Account, settings)
		location, err := url.Parse(response.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, settings, location.Path)
		require.Equal(t, "ok", location.Query().Get("sso_test"))
		require.Empty(t, location.Query().Get("reason"))
		saved, err := models.FindSSOConnectionByID(database.Conn(), conn.ID)
		require.NoError(t, err)
		require.NotNil(t, saved.LastTestedAt)
		require.NotContains(t, string(saved.LastTestResult), "eyJ")
		require.Contains(t, string(saved.LastTestResult), "owner@example.com")
		_, err = models.FindSSOIdentityBySubject(database.Conn(), conn.ID, fake.subject)
		require.Error(t, err)
	})

	t.Run("ignores an external redirect", func(t *testing.T) {
		response := f.testCallback(t, conn, fake, f.r.Account, "https://evil.example/phish")
		location := response.Header().Get("Location")
		require.NotContains(t, location, "evil.example")
		require.Contains(t, location, "/"+f.r.Organization.Slug+"/settings/sso")
		require.Contains(t, location, "sso_test=ok")
	})

	t.Run("stores a discovery failure", func(t *testing.T) {
		broken, _ := newFakeOIDC(t)
		broken.discoveryStatus = http.StatusInternalServerError
		broken.discoveryBody = "secret-issuer-body"
		input := connectionInput(true)
		input.Issuer = broken.issuer
		failed := f.put(input)
		response := f.testCallback(t, failed, broken, f.r.Account, settings)
		location, err := url.Parse(response.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, settings, location.Path)
		require.Equal(t, "failed", location.Query().Get("sso_test"))
		require.Equal(t, "begin", location.Query().Get("reason"))
		saved, err := models.FindSSOConnectionByID(database.Conn(), failed.ID)
		require.NoError(t, err)
		var result models.SSOTestResultDocument
		require.NoError(t, json.Unmarshal(saved.LastTestResult, &result))
		require.False(t, result.OK)
		require.Contains(t, result.Error, "SuperPlane could not read the identity provider configuration. Check the issuer URL.")
		require.Contains(t, result.Error, broken.issuer+"/.well-known/openid-configuration returned 500.")
		require.NotContains(t, result.Error, "secret-issuer-body")
	})
}

func (f *flow) testCallback(t *testing.T, conn *models.SSOConnection, fake *fakeOIDC, account *models.Account, redirect string) *httptest.ResponseRecorder {
	t.Helper()
	raw := "/auth/sso/" + conn.ID.String() + "/start?test=1"
	if redirect != "" {
		raw += "&redirect=" + url.QueryEscape(redirect)
	}
	start := httptest.NewRequest(http.MethodGet, raw, nil)
	if account != nil {
		token, err := authentication.GenerateAccountToken(f.service.Signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		start.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	}
	started := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(started, start)
	require.Equal(t, http.StatusFound, started.Code, started.Header().Get("Location"))
	location, err := url.Parse(started.Header().Get("Location"))
	require.NoError(t, err)
	if !strings.Contains(started.Header().Get("Location"), "/authorize") {
		return started
	}
	fake.nonce = location.Query().Get("nonce")
	callback := httptest.NewRequest(http.MethodGet, "/auth/sso/"+conn.ID.String()+"/callback?code=code-1&state="+url.QueryEscape(location.Query().Get("state")), nil)
	for _, cookie := range started.Result().Cookies() {
		callback.AddCookie(cookie)
	}
	if account != nil {
		token, err := authentication.GenerateAccountToken(f.service.Signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)
		callback.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	}
	response := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(response, callback)
	return response
}

type callbackOptions struct {
	state           string
	keepNonce       bool
	dropCookie      bool
	otherConnection bool
}

func (f *flow) callback(t *testing.T, conn *models.SSOConnection, fake *fakeOIDC, options callbackOptions) *httptest.ResponseRecorder {
	t.Helper()
	start := httptest.NewRequest(http.MethodGet, "/auth/sso/"+conn.ID.String()+"/start", nil)
	started := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(started, start)
	require.Equal(t, http.StatusFound, started.Code, started.Body.String())
	location, err := url.Parse(started.Header().Get("Location"))
	require.NoError(t, err)
	if !options.keepNonce {
		fake.nonce = location.Query().Get("nonce")
	}
	state := location.Query().Get("state")
	if options.state != "" {
		state = options.state
	}
	callback := httptest.NewRequest(http.MethodGet, "/auth/sso/"+conn.ID.String()+"/callback?code=code-1&state="+url.QueryEscape(state), nil)
	if !options.dropCookie {
		for _, cookie := range started.Result().Cookies() {
			if options.otherConnection && cookie.Name == loginStateCookie {
				token, err := f.service.Signer.GenerateWithClaims(10*time.Minute, map[string]string{
					"cid": "99999999-9999-4999-8999-999999999999", "st": state, "nonce": fake.nonce, "vf": "verifier", "rd": "", "test": "0",
				})
				require.NoError(t, err)
				cookie.Value = token
			}
			callback.AddCookie(cookie)
		}
	}
	response := httptest.NewRecorder()
	f.service.AuthHandler().ServeHTTP(response, callback)
	return response
}
