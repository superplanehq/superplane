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
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type fakeOIDC struct {
	issuer   string
	key      *rsa.PrivateKey
	nonce    string
	audience string
	expired  bool
	subject  string
	email    string
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
	return jwtLib.MapClaims{
		"iss": f.issuer, "sub": f.subject, "aud": f.audience,
		"exp": exp.Unix(), "iat": now.Unix(), "nonce": f.nonce,
		"email": f.email, "name": "Owner", "sid": "sid-1",
	}
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
