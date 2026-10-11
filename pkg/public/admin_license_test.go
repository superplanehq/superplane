package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
)

const licensePath = "/admin/api/installation/license"

func licenseBody(t *testing.T, raw []byte) []byte {
	t.Helper()
	body, err := json.Marshal(installLicenseRequest{License: string(raw)})
	require.NoError(t, err)
	return body
}

func decodeLicenseResponse(t *testing.T, body *bytes.Buffer) installationLicenseResponse {
	t.Helper()
	var response installationLicenseResponse
	require.NoError(t, json.Unmarshal(body.Bytes(), &response))
	return response
}

func TestAdminInstallationLicense(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	issuer := licensingtest.NewIssuer("test-key")
	server.SetLicenseService(licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		licensing.NewDatabaseSource(crypto.NewAESGCMEncryptor([]byte("0123456789abcdef0123456789abcdef"))),
	))

	t.Run("non-admin cannot read or change the license", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-license@example.com")
		require.NoError(t, err)
		regularToken, err := authentication.GenerateAccountToken(
			jwt.NewSigner("test-client-secret"), account.ID.String(), time.Now(), time.Hour,
		)
		require.NoError(t, err)

		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			response := execRequest(server, requestParams{
				method:      method,
				path:        licensePath,
				body:        licenseBody(t, issuer.License(licensing.FeatureGroups)),
				contentType: "application/json",
				authCookie:  regularToken,
			})
			assert.Equal(t, http.StatusNotFound, response.Code, method)
		}
	})

	t.Run("unauthenticated requests are rejected", func(t *testing.T) {
		response := execRequest(server, requestParams{method: http.MethodGet, path: licensePath})
		assert.NotEqual(t, http.StatusOK, response.Code)
	})

	t.Run("reports Community before a license is installed", func(t *testing.T) {
		response := execRequest(server, requestParams{method: http.MethodGet, path: licensePath, authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)

		status := decodeLicenseResponse(t, response.Body)
		assert.Equal(t, "community", status.Edition)
		assert.Equal(t, "none", status.State)
		assert.False(t, status.ManagedByConfiguration)
		assert.Nil(t, status.License)
	})

	t.Run("rejects an invalid license without changing state", func(t *testing.T) {
		other := licensingtest.NewIssuer("untrusted-key")
		response := execRequest(server, requestParams{
			method:      http.MethodPut,
			path:        licensePath,
			body:        licenseBody(t, other.License(licensing.FeatureGroups)),
			contentType: "application/json",
			authCookie:  token,
		})
		assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		assert.Contains(t, response.Body.String(), "does not trust the key")
		assert.Equal(t, "community", decodeLicenseResponse(t, execRequest(server, requestParams{
			method: http.MethodGet, path: licensePath, authCookie: token,
		}).Body).Edition)
	})

	t.Run("rejects an expired license", func(t *testing.T) {
		claims := licensingtest.Claims(licensing.FeatureGroups)
		claims["iat"] = time.Now().Add(-48 * time.Hour).Unix()
		claims["nbf"] = time.Now().Add(-48 * time.Hour).Unix()
		claims["exp"] = time.Now().Add(-24 * time.Hour).Unix()

		response := execRequest(server, requestParams{
			method:      http.MethodPut,
			path:        licensePath,
			body:        licenseBody(t, issuer.Sign(claims)),
			contentType: "application/json",
			authCookie:  token,
		})
		assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
		assert.Contains(t, response.Body.String(), "expired")
	})

	t.Run("rejects empty, unknown, and oversized bodies", func(t *testing.T) {
		cases := []struct {
			body []byte
			code int
		}{
			{body: []byte(`{"license":"   "}`), code: http.StatusBadRequest},
			{body: []byte(`{"license":"x","extra":true}`), code: http.StatusBadRequest},
			{body: []byte(`not json`), code: http.StatusBadRequest},
			{
				body: []byte(`{"license":"` + strings.Repeat("a", MaxLicenseRequestBytes) + `"}`),
				code: http.StatusRequestEntityTooLarge,
			},
		}

		for _, c := range cases {
			response := execRequest(server, requestParams{
				method:      http.MethodPut,
				path:        licensePath,
				body:        c.body,
				contentType: "application/json",
				authCookie:  token,
			})
			assert.Equal(t, c.code, response.Code)
		}
	})

	t.Run("installs a valid license and never returns it", func(t *testing.T) {
		raw := issuer.License(licensing.FeatureCustomRoles, licensing.FeatureGroups)
		response := execRequest(server, requestParams{
			method:      http.MethodPut,
			path:        licensePath,
			body:        licenseBody(t, raw),
			contentType: "application/json",
			authCookie:  token,
		})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		assert.NotContains(t, response.Body.String(), string(raw))

		status := decodeLicenseResponse(t, response.Body)
		assert.Equal(t, "enterprise", status.Edition)
		assert.Equal(t, "active", status.State)
		assert.Equal(t, "database", status.Source)
		require.NotNil(t, status.License)
		assert.ElementsMatch(t, []string{"custom_roles", "groups"}, status.License.Features)

		stored, err := models.FindInstallationLicense(database.Conn())
		require.NoError(t, err)
		assert.False(t, bytes.Contains(stored.EncryptedLicense, raw), "the license must be encrypted at rest")
	})

	t.Run("account response exposes entitlements", func(t *testing.T) {
		response := execRequest(server, requestParams{method: http.MethodGet, path: "/account", authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)

		var account AccountResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &account))
		assert.Equal(t, "enterprise", account.License.Edition)
		assert.ElementsMatch(t, []string{"custom_roles", "groups"}, account.License.Features)
		assert.Equal(t, "active", account.License.State)
		assert.NotNil(t, account.License.ExpiresAt)
	})

	t.Run("removes the license", func(t *testing.T) {
		response := execRequest(server, requestParams{method: http.MethodDelete, path: licensePath, authCookie: token})
		require.Equal(t, http.StatusOK, response.Code)

		status := decodeLicenseResponse(t, response.Body)
		assert.Equal(t, "community", status.Edition)
		assert.Equal(t, "none", status.State)
	})
}

func TestAdminInstallationLicenseManagedByConfiguration(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	server.SetLicenseService(licensingtest.EnterpriseService(licensing.FeatureGroups))
	issuer := licensingtest.NewIssuer("test-key")

	response := execRequest(server, requestParams{method: http.MethodGet, path: licensePath, authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	status := decodeLicenseResponse(t, response.Body)
	assert.True(t, status.ManagedByConfiguration)
	assert.Equal(t, "file", status.Source)

	response = execRequest(server, requestParams{
		method:      http.MethodPut,
		path:        licensePath,
		body:        licenseBody(t, issuer.License(licensing.FeatureGroups)),
		contentType: "application/json",
		authCookie:  token,
	})
	assert.Equal(t, http.StatusConflict, response.Code)

	response = execRequest(server, requestParams{method: http.MethodDelete, path: licensePath, authCookie: token})
	assert.Equal(t, http.StatusConflict, response.Code)
}

func TestAccountLicenseHidesStateFromNonAdmins(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	status := licensing.Status{
		Edition: licensing.EditionEnterprise,
		State:   licensing.StateActive,
		License: &licensing.License{
			Features:  []licensing.Feature{licensing.FeatureGroups},
			ExpiresAt: expiresAt,
		},
	}

	member := accountLicense(status, false)
	assert.Equal(t, "enterprise", member.Edition)
	assert.Equal(t, []string{"groups"}, member.Features)
	assert.Empty(t, member.State)
	assert.Nil(t, member.ExpiresAt)

	admin := accountLicense(status, true)
	assert.Equal(t, "active", admin.State)
	require.NotNil(t, admin.ExpiresAt)
	assert.False(t, admin.HideExpiryBanner)

	status.Edition = licensing.EditionCommunity
	status.State = licensing.StateExpired
	expired := accountLicense(status, false)
	assert.Equal(t, []string{}, expired.Features)
}

func TestAccountLicenseHidesExpiryBanner(t *testing.T) {
	status := licensing.Status{Edition: licensing.EditionCommunity, State: licensing.StateNone}

	t.Setenv(HideLicenseExpiryBannerEnv, "no")
	assert.False(t, accountLicense(status, true).HideExpiryBanner)

	t.Setenv(HideLicenseExpiryBannerEnv, "yes")
	assert.True(t, accountLicense(status, true).HideExpiryBanner)

	t.Setenv(HideLicenseExpiryBannerEnv, "true")
	assert.True(t, accountLicense(status, false).HideExpiryBanner)
}

func TestAdminLicenseKeyList(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	root := licensingtest.NewIssuer("root-key")
	first := licensingtest.NewIssuer("signing-key-1")
	next := licensingtest.NewIssuer("signing-key-2")
	store, err := licensing.NewKeyStore(licensingtest.KeySet(root), root.KeyList(1, first), nil)
	require.NoError(t, err)
	keySync := licensing.NewKeySync(store, licensing.DatabaseKeyListCache{}, "")
	server.SetLicenseService(licensing.NewService(
		licensing.NewVerifier(store),
		licensing.NewDatabaseSource(crypto.NewAESGCMEncryptor([]byte("0123456789abcdef0123456789abcdef"))),
		licensing.WithKeySync(keySync),
	))

	put := func(path string, body []byte, cookie string) *httptest.ResponseRecorder {
		return execRequest(server, requestParams{method: http.MethodPut, path: path, body: body, contentType: "application/json", authCookie: cookie})
	}
	keyListBody := func(raw []byte) []byte {
		body, err := json.Marshal(installKeyListRequest{KeyList: string(raw)})
		require.NoError(t, err)
		return body
	}

	status := decodeLicenseResponse(t, execRequest(server, requestParams{method: http.MethodGet, path: licensePath, authCookie: token}).Body)
	require.NotNil(t, status.TrustedKeys)
	assert.Equal(t, "disabled", status.TrustedKeys.State)
	assert.Equal(t, int64(1), status.TrustedKeys.Version)

	response := put(licensePath, licenseBody(t, next.License(licensing.FeatureGroups)), token)
	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)

	account, err := models.CreateAccount("Regular User", "regular-keys@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, put(licensePath+"/keys", keyListBody(root.KeyList(2, next)), regularToken).Code)

	response = put(licensePath+"/keys", keyListBody(licensingtest.NewIssuer("root-key").KeyList(2, next)), token)
	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "not valid")

	response = put(licensePath+"/keys", keyListBody(root.KeyList(2, first, next)), token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	status = decodeLicenseResponse(t, response.Body)
	assert.Equal(t, int64(2), status.TrustedKeys.Version)
	assert.Equal(t, "synced", status.TrustedKeys.State)

	response = put(licensePath, licenseBody(t, next.License(licensing.FeatureGroups)), token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, "active", decodeLicenseResponse(t, response.Body).State)

	response = put(licensePath+"/keys", keyListBody(root.KeyList(1, first)), token)
	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "older")

	assert.Equal(t, http.StatusBadRequest, put(licensePath+"/keys", []byte(`{"key_list":" "}`), token).Code)
}
