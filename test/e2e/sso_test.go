package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	jwtLib "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func TestSSOSettingsPage(t *testing.T) {
	s := ctx.NewSession(t)
	s.Start()
	require.NoError(t, models.EnableExperimentalFeature(s.OrgID, features.FeatureFactories))
	s.Login()
	s.Visit("/" + s.OrgID.String() + "/settings/sso")
	s.AssertText("Single Sign-On")
	s.AssertText("Identity provider")
}

func TestSSOConfigureTestAndSignIn(t *testing.T) {
	s := ctx.NewSession(t)
	s.Start()
	t.Cleanup(func() {
		clearSSO(t, s)
		if member, err := models.FindActiveUserByEmail(s.OrgID.String(), "ada@superplane.local"); err == nil {
			_ = member.SoftDelete(database.Conn(), time.Now(), "")
		}
		_ = models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
			metadata.AllowPrivateNetworkAccess = false
		})
	})
	clearSSO(t, s)
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	time.Sleep(6 * time.Second)
	fake := newE2EOIDC(t, "ada@superplane.local")
	require.NoError(t, models.EnableExperimentalFeature(s.OrgID, features.FeatureFactories))
	s.Login()
	s.Visit("/" + s.OrgID.String() + "/settings/sso")
	s.Click(queries.Locator("button:text-is(\"Create connection\")"))
	s.FillIn(queries.Locator("#sso-issuer"), fake.issuer)
	s.FillIn(queries.Locator("#sso-client-id"), "client-1")
	s.FillIn(queries.Locator("#sso-client-secret"), "secret")
	s.Click(queries.TestID("sso-provider-save"))
	s.AssertText("Connection saved.")
	s.FillIn(queries.Locator("#sso-domain"), "superplane.local")
	s.Click(queries.Locator("button:text-is(\"Add domain\")"))
	s.AssertText("Pending")

	var domain models.OrganizationDomain
	require.NoError(t, database.Conn().Where("organization_id = ? AND domain = ?", s.OrgID, "superplane.local").First(&domain).Error)
	now := time.Now()
	domain.VerifiedAt = &now
	method := models.DomainVerificationDNS
	domain.VerificationMethod = &method
	require.NoError(t, database.Conn().Save(&domain).Error)
	s.Visit("/" + s.OrgID.String() + "/settings/sso")
	s.AssertText("Verified")
	s.Click(queries.Locator("button:text-is(\"Test connection\")"))
	s.AssertText("The sign-in test succeeded.")
	s.Click(queries.Locator("button:text-is(\"Activate\")"))
	s.AssertText("Single Sign-On is active.")

	require.NoError(t, s.Page().Context().ClearCookies())
	s.Visit("/login")
	s.Click(queries.Text("Continue with SSO"))
	s.FillIn(queries.Locator("#sso-email"), "ada@superplane.local")
	s.Click(queries.Text("Continue with SSO"))
	require.Eventually(t, func() bool {
		return strings.Contains(s.Page().URL(), "/"+s.OrgSlug)
	}, 20*time.Second, 200*time.Millisecond)

	s.Login()
	s.Visit("/" + s.OrgID.String() + "/settings/members")
	s.AssertText("ada@superplane.local")
	s.AssertText("SSO")

	conn, err := models.FindSSOConnectionByOrganization(database.Conn(), s.OrgID)
	require.NoError(t, err)
	conn.Enforcement = models.SSOEnforcementRequired
	conn.AllowOwnerBypass = false
	require.NoError(t, conn.Save(database.Conn()))
	s.Login()
	s.Visit("/" + s.OrgID.String() + "/settings/members")
	require.Eventually(t, func() bool {
		return accountCookieHasSSO(t, s)
	}, 20*time.Second, 200*time.Millisecond)
}

func accountCookieHasSSO(t *testing.T, s *session.TestSession) bool {
	t.Helper()
	cookies, err := s.Page().Context().Cookies()
	if err != nil {
		return false
	}
	for _, cookie := range cookies {
		if cookie.Name != "account_token" {
			continue
		}
		parts := strings.Split(cookie.Value, ".")
		if len(parts) < 2 {
			return false
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
		return strings.Contains(string(payload), `"sso"`)
	}
	return false
}

func TestSSORequiredEnforcementRedirects(t *testing.T) {
	s := ctx.NewSession(t)
	s.Start()
	t.Cleanup(func() { clearSSO(t, s) })
	clearSSO(t, s)
	connection := &models.SSOConnection{
		OrganizationID: s.OrgID, Protocol: models.SSOProtocolOIDC, Preset: models.SSOPresetGeneric,
		DisplayName: "OIDC", Status: models.SSOStatusActive, Enforcement: models.SSOEnforcementRequired,
		AllowOwnerBypass: false, JITEnabled: true, DefaultRole: models.RoleOrgOperator,
		MaxSessionAgeSeconds: 86400, Config: []byte(`{"issuer":"https://idp.example.com","client_id":"client-1"}`),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, database.Conn().Create(connection).Error)

	request, err := http.NewRequest(http.MethodGet, s.BaseURL+"/api/v1/users", nil)
	require.NoError(t, err)
	request.Header.Set("x-organization-id", s.OrgID.String())
	request.AddCookie(accountCookie(t, s))
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
	require.Contains(t, response.Header.Get("X-SuperPlane-SSO-Start"), connection.ID.String())
}

func TestSSOSignInCreatesMember(t *testing.T) {
	s := ctx.NewSession(t)
	s.Start()
	t.Cleanup(func() {
		clearSSO(t, s)
		if member, err := models.FindActiveUserByEmail(s.OrgID.String(), "ada@superplane.local"); err == nil {
			_ = member.SoftDelete(database.Conn(), time.Now(), "")
		}
	})
	clearSSO(t, s)
	require.NoError(t, models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
		metadata.AllowPrivateNetworkAccess = true
	}))
	t.Cleanup(func() {
		_ = models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
			metadata.AllowPrivateNetworkAccess = false
		})
	})

	fake := newE2EOIDC(t, "ada@superplane.local")
	client := authenticatedClient(t, s)
	createDomain(t, s, client, "superplane.local")
	var domain models.OrganizationDomain
	require.NoError(t, database.Conn().Where("organization_id = ? AND domain = ?", s.OrgID, "superplane.local").First(&domain).Error)
	now := time.Now()
	domain.VerifiedAt = &now
	method := models.DomainVerificationDNS
	domain.VerificationMethod = &method
	require.NoError(t, database.Conn().Save(&domain).Error)

	connection := putConnection(t, s, client, fake.issuer)
	start := getNoRedirect(t, client, s.BaseURL+"/auth/sso/"+connection+"/start?test=1&login_hint=ada@superplane.local")
	location, err := url.Parse(start.Header.Get("Location"))
	require.NoError(t, err)
	fake.nonce = location.Query().Get("nonce")
	callback := getNoRedirect(t, client, s.BaseURL+"/auth/sso/"+connection+"/callback?code=code-1&state="+url.QueryEscape(location.Query().Get("state")))
	require.Contains(t, callback.Header.Get("Location"), "sso_test=ok")

	activate := postJSON(t, s, client, "/api/v1/sso/connection/activate", nil)
	require.Equal(t, http.StatusOK, activate.StatusCode)
	activate.Body.Close()

	guest := guestClient(t)
	signIn := getNoRedirect(t, guest, s.BaseURL+"/auth/sso/"+connection+"/start?login_hint=ada@superplane.local")
	signInLocation, err := url.Parse(signIn.Header.Get("Location"))
	require.NoError(t, err)
	fake.nonce = signInLocation.Query().Get("nonce")
	done := getNoRedirect(t, guest, s.BaseURL+"/auth/sso/"+connection+"/callback?code=code-1&state="+url.QueryEscape(signInLocation.Query().Get("state")))
	require.Contains(t, done.Header.Get("Location"), "/"+s.OrgSlug)

	member, err := models.FindActiveUserByEmail(s.OrgID.String(), "ada@superplane.local")
	require.NoError(t, err)
	require.NotNil(t, member.ProvisionedBy)
	require.Equal(t, models.SSOProvisionedByJIT, *member.ProvisionedBy)
}

func clearSSO(t *testing.T, s *session.TestSession) {
	t.Helper()
	require.NoError(t, database.Conn().Where("organization_id = ?", s.OrgID).Delete(&models.SSOConnection{}).Error)
	require.NoError(t, database.Conn().Where("organization_id = ?", s.OrgID).Delete(&models.OrganizationDomain{}).Error)
}

func accountCookie(t *testing.T, s *session.TestSession) *http.Cookie {
	t.Helper()
	signer := spjwt.NewSigner(os.Getenv("JWT_SECRET"))
	token, err := authentication.GenerateAccountToken(signer, s.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	return &http.Cookie{Name: "account_token", Value: token}
}

func authenticatedClient(t *testing.T, s *session.TestSession) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	base, err := url.Parse(s.BaseURL)
	require.NoError(t, err)
	jar.SetCookies(base, []*http.Cookie{accountCookie(t, s)})
	return &http.Client{Jar: jar, CheckRedirect: stopRedirect}
}

func guestClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &http.Client{Jar: jar, CheckRedirect: stopRedirect}
}

func stopRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func createDomain(t *testing.T, s *session.TestSession, client *http.Client, domain string) {
	t.Helper()
	response := postJSON(t, s, client, "/api/v1/sso/domains", map[string]any{"domain": domain})
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	require.Equalf(t, http.StatusOK, response.StatusCode, "%s", body)
}

func putConnection(t *testing.T, s *session.TestSession, client *http.Client, issuer string) string {
	t.Helper()
	response := putJSON(t, s, client, "/api/v1/sso/connection", map[string]any{
		"preset": "PRESET_GENERIC", "displayName": "OIDC", "issuer": issuer, "clientId": "client-1",
		"clientSecret": "secret", "jitEnabled": true, "defaultRole": "org_operator",
		"enforcement": "ENFORCEMENT_OPTIONAL", "allowOwnerBypass": true, "maxSessionAgeSeconds": 86400,
	})
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	require.Equalf(t, http.StatusOK, response.StatusCode, "%s", body)
	var payload struct {
		Connection struct {
			ID string `json:"id"`
		} `json:"connection"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NotEmpty(t, payload.Connection.ID)
	return payload.Connection.ID
}

func postJSON(t *testing.T, s *session.TestSession, client *http.Client, path string, payload any) *http.Response {
	t.Helper()
	return sendJSON(t, s, client, http.MethodPost, path, payload)
}

func putJSON(t *testing.T, s *session.TestSession, client *http.Client, path string, payload any) *http.Response {
	t.Helper()
	return sendJSON(t, s, client, http.MethodPut, path, payload)
}

func sendJSON(t *testing.T, s *session.TestSession, client *http.Client, method string, path string, payload any) *http.Response {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, s.BaseURL+path, body)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-organization-id", s.OrgID.String())
	response, err := client.Do(request)
	require.NoError(t, err)
	return response
}

func getNoRedirect(t *testing.T, client *http.Client, rawURL string) *http.Response {
	t.Helper()
	response, err := client.Get(rawURL)
	require.NoError(t, err)
	response.Body.Close()
	require.Equalf(t, http.StatusFound, response.StatusCode, "location %s", response.Header.Get("Location"))
	return response
}

type e2eOIDC struct {
	issuer string
	key    *rsa.PrivateKey
	nonce  string
}

func newE2EOIDC(t *testing.T, email string) *e2eOIDC {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	fake := &e2eOIDC{key: key}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(w, map[string]any{
				"issuer": fake.issuer, "authorization_endpoint": fake.issuer + "/authorize",
				"token_endpoint": fake.issuer + "/token", "jwks_uri": fake.issuer + "/jwks",
				"userinfo_endpoint": fake.issuer + "/userinfo", "end_session_endpoint": fake.issuer + "/logout",
				"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/authorize":
			fake.nonce = r.URL.Query().Get("nonce")
			target := r.URL.Query().Get("redirect_uri") + "?code=code-1&state=" + url.QueryEscape(r.URL.Query().Get("state"))
			http.Redirect(w, r, target, http.StatusFound)
		case "/jwks":
			writeJSON(w, map[string]any{"keys": []any{map[string]string{
				"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "test",
				"n": base64.RawURLEncoding.EncodeToString(fake.key.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(fake.key.E)).Bytes()),
			}}})
		case "/token":
			claims := jwtLib.MapClaims{
				"iss": fake.issuer, "sub": "ada", "aud": "client-1", "nonce": fake.nonce,
				"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
				"email": email, "name": "Ada",
			}
			token := jwtLib.NewWithClaims(jwtLib.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			signed, signErr := token.SignedString(fake.key)
			if signErr != nil {
				http.Error(w, signErr.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"access_token": "access", "token_type": "Bearer", "id_token": signed})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	fake.issuer = server.URL
	return fake
}
