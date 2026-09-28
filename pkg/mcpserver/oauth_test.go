package mcpserver

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const testPKCEVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func testPKCEChallenge() string {
	sum := sha256.Sum256([]byte(testPKCEVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func insertAuthCode(t *testing.T, resource string) string {
	t.Helper()
	raw := "one-time-code-" + uuid.NewString()
	err := models.CreateMCPOAuthCode(database.Conn(), &models.MCPOAuthCode{
		CodeHash:            models.HashMCPOAuthSecret(raw),
		ClientID:            LocalClientID,
		RedirectURI:         CursorRedirectURIs[0],
		Resource:            resource,
		CodeChallenge:       testPKCEChallenge(),
		CodeChallengeMethod: "S256",
		UserID:              uuid.New(),
		OrganizationID:      uuid.New(),
		FactoryID:           uuid.New(),
		Scopes:              datatypes.NewJSONSlice(GrantedScopes),
		ExpiresAt:           time.Now().Add(time.Minute),
	})
	require.NoError(t, err)
	return raw
}

func tokenForm(code, verifier, resource string) url.Values {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("code_verifier", verifier)
	values.Set("client_id", LocalClientID)
	values.Set("redirect_uri", CursorRedirectURIs[0])
	if resource != "" {
		values.Set("resource", resource)
	}
	return values
}

func TestExchangeAuthorizationCodeRequiresPKCE(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, "", resource), resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_request", oauthErr.Code)
		return nil
	})
	require.NoError(t, err)
}

func TestExchangeAuthorizationCodeRejectsWrongResource(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, testPKCEVerifier, "https://other.example/mcp"), resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_target", oauthErr.Code)
		return nil
	})
	require.NoError(t, err)
}

func TestExchangeAuthorizationCodeRejectsReusedCode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		issue, refresh, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, testPKCEVerifier, resource), resource)
		require.Nil(t, oauthErr)
		require.NotNil(t, issue)
		require.NotEmpty(t, refresh)
		return nil
	})
	require.NoError(t, err)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, testPKCEVerifier, resource), resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		return nil
	})
	require.NoError(t, err)
}

func TestRegisterClientAllowsCursorRedirects(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	client, oauthErr := RegisterClient(database.Conn(), []byte(`{"client_name":"Cursor","redirect_uris":["cursor://anysphere.cursor-mcp/oauth/callback"]}`))
	require.Nil(t, oauthErr)
	require.Contains(t, client.RedirectURIs, "http://localhost:8787/callback")
	require.Contains(t, client.RedirectURIs, "cursor://anysphere.cursor-mcp/oauth/callback")
}

func TestParseAuthorizeRequestRequiresPKCE(t *testing.T) {
	_, err := ParseAuthorizeRequest(url.Values{
		"client_id":     {LocalClientID},
		"redirect_uri":  {CursorRedirectURIs[0]},
		"response_type": {"code"},
	}, "http://localhost:8000/mcp")
	require.NotNil(t, err)
	require.Equal(t, "invalid_request", err.Code)
}

func TestParseAuthorizeRequestRejectsWrongResource(t *testing.T) {
	_, err := ParseAuthorizeRequest(url.Values{
		"client_id":             {LocalClientID},
		"redirect_uri":          {CursorRedirectURIs[0]},
		"response_type":         {"code"},
		"code_challenge":        {testPKCEChallenge()},
		"code_challenge_method": {"S256"},
		"resource":              {"https://other.example/mcp"},
	}, "http://localhost:8000/mcp")
	require.NotNil(t, err)
	require.Equal(t, "invalid_target", err.Code)
}
