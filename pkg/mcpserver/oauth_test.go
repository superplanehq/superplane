package mcpserver

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const testPKCEVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func insertAuthCode(t *testing.T, r *support.ResourceRegistry, resource string) string {
	t.Helper()
	raw := "one-time-code-" + uuid.NewString()
	err := models.CreateMCPOAuthCode(database.Conn(), &models.MCPOAuthCode{
		CodeHash:            crypto.HashToken(raw),
		ClientID:            LocalClientID,
		RedirectURI:         CursorRedirectURIs[0],
		Resource:            resource,
		CodeChallenge:       mcp.S256Challenge(testPKCEVerifier),
		CodeChallengeMethod: "S256",
		UserID:              r.User,
		OrganizationID:      r.Organization.ID,
		FactoryID:           uuid.New(),
		Scopes:              datatypes.NewJSONSlice(GrantedScopes),
		ExpiresAt:           time.Now().Add(time.Minute),
	})
	require.NoError(t, err)
	return raw
}

func insertRefreshToken(t *testing.T, r *support.ResourceRegistry, resource string) string {
	t.Helper()
	raw := "refresh-" + uuid.NewString()
	err := models.CreateMCPOAuthRefreshToken(database.Conn(), &models.MCPOAuthRefreshToken{
		TokenHash:      crypto.HashToken(raw),
		ClientID:       LocalClientID,
		UserID:         r.User,
		OrganizationID: r.Organization.ID,
		FactoryID:      uuid.New(),
		Resource:       resource,
		Scopes:         datatypes.NewJSONSlice(GrantedScopes),
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	return raw
}

func countAuthCodes(t *testing.T, tx *gorm.DB, raw string) int64 {
	t.Helper()
	var count int64
	err := tx.Model(&models.MCPOAuthCode{}).Where("code_hash = ?", crypto.HashToken(raw)).Count(&count).Error
	require.NoError(t, err)
	return count
}

func countRefreshTokens(t *testing.T, tx *gorm.DB, raw string) int64 {
	t.Helper()
	var count int64
	err := tx.Model(&models.MCPOAuthRefreshToken{}).Where("token_hash = ?", crypto.HashToken(raw)).Count(&count).Error
	require.NoError(t, err)
	return count
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
	code := insertAuthCode(t, r, resource)

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
	code := insertAuthCode(t, r, resource)

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
	code := insertAuthCode(t, r, resource)

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

func TestResolveClientAllowsEphemeralLoopbackPort(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	tests := []struct {
		name       string
		registered string
		requested  string
		allowed    bool
	}{
		{
			name:       "ipv4 ephemeral port",
			registered: "http://127.0.0.1/callback",
			requested:  "http://127.0.0.1:52291/callback",
			allowed:    true,
		},
		{
			name:       "localhost ephemeral port",
			registered: "http://localhost/callback",
			requested:  "http://localhost:52291/callback",
			allowed:    true,
		},
		{
			name:       "ipv6 ephemeral port",
			registered: "http://[::1]/callback",
			requested:  "http://[::1]:52291/callback",
			allowed:    true,
		},
		{
			name:       "exact redirect",
			registered: "https://app.example/callback",
			requested:  "https://app.example/callback",
			allowed:    true,
		},
		{
			name:       "different path",
			registered: "http://127.0.0.1/callback",
			requested:  "http://127.0.0.1:52291/other",
			allowed:    false,
		},
		{
			name:       "different query",
			registered: "http://127.0.0.1/callback",
			requested:  "http://127.0.0.1:52291/callback?extra=1",
			allowed:    false,
		},
		{
			name:       "different host",
			registered: "http://127.0.0.1/callback",
			requested:  "http://127.0.0.2:52291/callback",
			allowed:    false,
		},
		{
			name:       "userinfo",
			registered: "http://127.0.0.1/callback",
			requested:  "http://user@127.0.0.1:52291/callback",
			allowed:    false,
		},
		{
			name:       "fragment",
			registered: "http://127.0.0.1/callback",
			requested:  "http://127.0.0.1:52291/callback#code",
			allowed:    false,
		},
		{
			name:       "non-loopback port change",
			registered: "https://app.example/callback",
			requested:  "https://app.example:8443/callback",
			allowed:    false,
		},
		{
			name:       "https loopback port change",
			registered: "https://127.0.0.1/callback",
			requested:  "https://127.0.0.1:8443/callback",
			allowed:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientID := uuid.NewString()
			_, err := models.CreateMCPOAuthClient(database.Conn(), clientID, "OpenCode", []string{test.registered})
			require.NoError(t, err)

			client, oauthErr := ResolveClient(t.Context(), database.Conn(), nil, clientID, test.requested)
			if !test.allowed {
				require.Nil(t, client)
				require.NotNil(t, oauthErr)
				require.Equal(t, "redirect_uri is not allowed", oauthErr.Error())
				return
			}
			require.Nil(t, oauthErr)
			require.NotNil(t, client)
			require.Equal(t, clientID, client.ID)
		})
	}
}

func TestResolveClientKeepsCursorCallbackExact(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	_, oauthErr := ResolveClient(t.Context(), database.Conn(), nil, LocalClientID, "http://localhost:52291/callback")
	require.NotNil(t, oauthErr)
	require.Equal(t, "redirect_uri is not allowed", oauthErr.Error())

	client, oauthErr := ResolveClient(t.Context(), database.Conn(), nil, LocalClientID, CursorRedirectURIs[0])
	require.Nil(t, oauthErr)
	require.Equal(t, LocalClientID, client.ID)

	registered, oauthErr := RegisterClient(database.Conn(), []byte(`{"client_name":"Cursor","redirect_uris":["cursor://anysphere.cursor-mcp/oauth/callback"]}`))
	require.Nil(t, oauthErr)
	_, oauthErr = ResolveClient(t.Context(), database.Conn(), nil, registered.ID, "http://localhost:52291/callback")
	require.NotNil(t, oauthErr)
	require.Equal(t, "redirect_uri is not allowed", oauthErr.Error())
}

func TestRegisterClientAllowsCursorRedirects(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	client, oauthErr := RegisterClient(database.Conn(), []byte(`{"client_name":"Cursor","redirect_uris":["cursor://anysphere.cursor-mcp/oauth/callback"]}`))
	require.Nil(t, oauthErr)
	require.Contains(t, client.RedirectURIs, "http://localhost:8787/callback")
	require.Contains(t, client.RedirectURIs, "cursor://anysphere.cursor-mcp/oauth/callback")
}

func TestClientDisplayNameUsesCodexForMetadataClientID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const codexID = "https://chatgpt.com/oauth/codex/example/client.json"
	db := database.Conn()
	require.Equal(t, "Codex", ClientDisplayName(db, codexID))
	require.Equal(t, models.DefaultMCPClientName, ClientDisplayName(db, "https://example.com/oauth/client.json"))
	require.Equal(t, LocalClientName, ClientDisplayName(db, LocalClientID))
	require.Equal(t, models.DefaultMCPClientName, ClientDisplayName(db, ""))

	_, err := models.CreateMCPOAuthClient(db, codexID, "Cursor Desktop", []string{"cursor://callback"})
	require.NoError(t, err)
	require.Equal(t, "Cursor Desktop", ClientDisplayName(db, codexID))

	storedURL := "https://www.chatgpt.com/oauth/codex/stored/client.json"
	_, err = models.CreateMCPOAuthClient(db, storedURL, storedURL, []string{"cursor://callback"})
	require.NoError(t, err)
	require.Equal(t, "Codex", ClientDisplayName(db, storedURL))
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
		"code_challenge":        {mcp.S256Challenge(testPKCEVerifier)},
		"code_challenge_method": {"S256"},
		"resource":              {"https://other.example/mcp"},
	}, "http://localhost:8000/mcp")
	require.NotNil(t, err)
	require.Equal(t, "invalid_target", err.Code)
}

func TestExchangeAuthorizationCodeKeepsCodeWhenPKCEFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, r, resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, "wrong-verifier", resource), resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		require.Equal(t, int64(1), countAuthCodes(t, tx, code))

		issue, refresh, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, testPKCEVerifier, resource), resource)
		require.Nil(t, oauthErr)
		require.NotNil(t, issue)
		require.NotEmpty(t, refresh)
		return nil
	})
	require.NoError(t, err)
}

func TestExchangeAuthorizationCodeKeepsCodeWhenClientMismatches(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, r, resource)
	values := tokenForm(code, testPKCEVerifier, resource)
	values.Set("client_id", "other-client")

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, values, resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		require.Equal(t, int64(1), countAuthCodes(t, tx, code))
		return nil
	})
	require.NoError(t, err)
}

func TestExchangeAuthorizationCodeRejectsBlockedAccount(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	code := insertAuthCode(t, r, resource)
	require.NoError(t, r.Account.Block(database.Conn(), time.Now()))

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := ExchangeAuthorizationCode(tx, tokenForm(code, testPKCEVerifier, resource), resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		require.Equal(t, int64(0), countAuthCodes(t, tx, code))
		return nil
	})
	require.NoError(t, err)
}

func TestRefreshTokensKeepsTokenWhenClientMismatches(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	raw := insertRefreshToken(t, r, resource)
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", raw)
	values.Set("client_id", "other-client")
	values.Set("resource", resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := RefreshTokens(tx, values, resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		require.Equal(t, int64(1), countRefreshTokens(t, tx, raw))
		return nil
	})
	require.NoError(t, err)
}

func TestRefreshTokensRejectsBlockedAccount(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	resource := "http://localhost:8000/mcp"
	raw := insertRefreshToken(t, r, resource)
	require.NoError(t, r.Account.Block(database.Conn(), time.Now()))
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", raw)
	values.Set("client_id", LocalClientID)
	values.Set("resource", resource)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, oauthErr := RefreshTokens(tx, values, resource)
		require.NotNil(t, oauthErr)
		require.Equal(t, "invalid_grant", oauthErr.Code)
		require.Equal(t, int64(0), countRefreshTokens(t, tx, raw))
		return nil
	})
	require.NoError(t, err)
}
