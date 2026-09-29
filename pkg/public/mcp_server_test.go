package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/mcpserver"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

const mcpTestPKCEVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func mcpEnabledServer(t *testing.T) (*support.ResourceRegistry, *Server, *jwt.Signer) {
	t.Helper()
	r := support.Setup(t)
	server, _, _ := setupTestServer(r, t)
	return r, server, jwt.NewSigner("test-client-secret")
}

func mcpRequest(method, path string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost:8000"
	if body != "" && !strings.HasPrefix(path, "/oauth/token") {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func mcpAccessClaims(r *support.ResourceRegistry, factoryID uuid.UUID) mcpserver.AccessClaims {
	return mcpserver.AccessClaims{
		UserID:    r.User,
		OrgID:     r.Organization.ID,
		FactoryID: factoryID,
		ClientID:  mcpserver.LocalClientID,
		Resource:  "http://localhost:8000/mcp",
		Scopes:    mcpserver.GrantedScopes,
	}
}

func insertMCPAccessGrant(t *testing.T, claims mcpserver.AccessClaims) {
	t.Helper()
	require.NoError(t, models.CreateMCPOAuthRefreshToken(database.DB(t.Context()), &models.MCPOAuthRefreshToken{
		TokenHash:      uuid.NewString(),
		ClientID:       claims.ClientID,
		UserID:         claims.UserID,
		OrganizationID: claims.OrgID,
		FactoryID:      claims.FactoryID,
		Resource:       claims.Resource,
		Scopes:         datatypes.NewJSONSlice(claims.Scopes),
		ExpiresAt:      time.Now().Add(time.Hour),
	}))
}

func TestMCPUnauthenticatedReturns401AndMetadataURL(t *testing.T) {
	_, server, _ := mcpEnabledServer(t)
	req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "http://localhost:8000/.well-known/oauth-protected-resource/mcp")
}

func TestMCPTokenExchangeFailureCases(t *testing.T) {
	r, server, _ := mcpEnabledServer(t)
	resource := "http://localhost:8000/mcp"
	code := "one-time-code-" + uuid.NewString()
	require.NoError(t, models.CreateMCPOAuthCode(database.Conn(), &models.MCPOAuthCode{
		CodeHash:            crypto.HashToken(code),
		ClientID:            mcpserver.LocalClientID,
		RedirectURI:         mcpserver.CursorRedirectURIs[0],
		Resource:            resource,
		CodeChallenge:       mcp.S256Challenge(mcpTestPKCEVerifier),
		CodeChallengeMethod: "S256",
		UserID:              r.User,
		OrganizationID:      r.Organization.ID,
		FactoryID:           uuid.New(),
		Scopes:              datatypes.NewJSONSlice(mcpserver.GrantedScopes),
		ExpiresAt:           time.Now().Add(time.Minute),
	}))

	postToken := func(values url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(values.Encode()))
		req.Host = "localhost:8000"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("missing PKCE", func(t *testing.T) {
		values := url.Values{}
		values.Set("grant_type", "authorization_code")
		values.Set("code", code)
		values.Set("client_id", mcpserver.LocalClientID)
		values.Set("redirect_uri", mcpserver.CursorRedirectURIs[0])
		values.Set("resource", resource)
		rec := postToken(values)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_request")
	})

	t.Run("wrong resource", func(t *testing.T) {
		values := url.Values{}
		values.Set("grant_type", "authorization_code")
		values.Set("code", code)
		values.Set("code_verifier", mcpTestPKCEVerifier)
		values.Set("client_id", mcpserver.LocalClientID)
		values.Set("redirect_uri", mcpserver.CursorRedirectURIs[0])
		values.Set("resource", "https://other.example/mcp")
		rec := postToken(values)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
	})

	t.Run("reused code", func(t *testing.T) {
		values := url.Values{}
		values.Set("grant_type", "authorization_code")
		values.Set("code", code)
		values.Set("code_verifier", mcpTestPKCEVerifier)
		values.Set("client_id", mcpserver.LocalClientID)
		values.Set("redirect_uri", mcpserver.CursorRedirectURIs[0])
		values.Set("resource", resource)

		first := postToken(values)
		assert.Equal(t, http.StatusOK, first.Code)

		second := postToken(values)
		assert.Equal(t, http.StatusBadRequest, second.Code)
		assert.Contains(t, second.Body.String(), "invalid_grant")
	})
}

func TestMCPToolCallWorkspaceBindingAndMissingAgent(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())

	factoryA, err := models.CreateFactory(db, r.Organization.ID, "Workspace A", "", "WSA")
	require.NoError(t, err)
	factoryB, err := models.CreateFactory(db, r.Organization.ID, "Workspace B", "", "WSB")
	require.NoError(t, err)
	orderB, err := factoryB.CreateWorkOrder(db, "Secret", "", &r.User, nil, nil)
	require.NoError(t, err)
	orderA, err := factoryA.CreateWorkOrder(db, "No agent", "", &r.User, nil, nil)
	require.NoError(t, err)

	token, err := mcpserver.MintAccessToken(signer, mcpAccessClaims(r, factoryA.ID), time.Hour)
	require.NoError(t, err)
	insertMCPAccessGrant(t, mcpAccessClaims(r, factoryA.ID))

	callTool := func(name string, args map[string]any) *httptest.ResponseRecorder {
		payload, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params":  map[string]any{"name": name, "arguments": args},
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(payload))
		req.Host = "localhost:8000"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		return rec
	}

	t.Run("other workspace is not found", func(t *testing.T) {
		rec := callTool("get_task", map[string]any{"task": orderB.ID.String()})
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		result, ok := body["result"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, result["isError"])
		content, ok := result["content"].([]any)
		require.True(t, ok)
		require.NotEmpty(t, content)
		item, ok := content[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Not found", item["text"])
	})

	t.Run("missing agent is a tool error", func(t *testing.T) {
		rec := callTool("get_task_agent", map[string]any{"task": orderA.ID.String()})
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Nil(t, body["error"])
		result, ok := body["result"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, result["isError"])
		content, ok := result["content"].([]any)
		require.True(t, ok)
		item, ok := content[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "This task has no agent session.", item["text"])
	})
}

func TestMCPBlockedAccountReturns401(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))

	token, err := mcpserver.MintAccessToken(signer, mcpAccessClaims(r, uuid.New()), time.Hour)
	require.NoError(t, err)
	require.NoError(t, r.Account.Block(database.DB(t.Context()), time.Now()))

	req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "http://localhost:8000/.well-known/oauth-protected-resource/mcp")
}

func TestMCPAuthenticatedWithoutFeatureFlagReturns404(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	token, err := mcpserver.MintAccessToken(signer, mcpAccessClaims(r, uuid.New()), time.Hour)
	require.NoError(t, err)

	req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMCPDiscoveryListsAreEmpty(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))

	claims := mcpAccessClaims(r, uuid.New())
	insertMCPAccessGrant(t, claims)
	token, err := mcpserver.MintAccessToken(signer, claims, time.Hour)
	require.NoError(t, err)

	for _, method := range []string{"resources/list", "prompts/list", "resources/templates/list"} {
		req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"`+method+`"}`)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, method)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), method)
		assert.Nil(t, body["error"], method)
		result, ok := body["result"].(map[string]any)
		require.True(t, ok, method)
		switch method {
		case "resources/list":
			assert.Equal(t, []any{}, result["resources"])
		case "prompts/list":
			assert.Equal(t, []any{}, result["prompts"])
		case "resources/templates/list":
			assert.Equal(t, []any{}, result["resourceTemplates"])
		}
	}
}

func TestMCPRevokedGrantReturns401(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	claims := mcpAccessClaims(r, factory.ID)
	insertMCPAccessGrant(t, claims)
	token, err := mcpserver.MintAccessToken(signer, claims, time.Hour)
	require.NoError(t, err)

	req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, models.DeleteMCPOAuthRefreshTokensForClient(
		database.DB(t.Context()),
		claims.OrgID,
		claims.FactoryID,
		claims.UserID,
		claims.ClientID,
	))
	rec = httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMCPRevokedPendingCodeDoesNotRestoreAccess(t *testing.T) {
	r, server, signer := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	claims := mcpAccessClaims(r, factory.ID)
	insertMCPAccessGrant(t, claims)
	token, err := mcpserver.MintAccessToken(signer, claims, time.Hour)
	require.NoError(t, err)

	code := "pending-code-" + uuid.NewString()
	require.NoError(t, models.CreateMCPOAuthCode(db, &models.MCPOAuthCode{
		CodeHash:            crypto.HashToken(code),
		ClientID:            claims.ClientID,
		RedirectURI:         mcpserver.CursorRedirectURIs[0],
		Resource:            claims.Resource,
		CodeChallenge:       mcp.S256Challenge(mcpTestPKCEVerifier),
		CodeChallengeMethod: "S256",
		UserID:              claims.UserID,
		OrganizationID:      claims.OrgID,
		FactoryID:           claims.FactoryID,
		Scopes:              datatypes.NewJSONSlice(claims.Scopes),
		ExpiresAt:           time.Now().Add(time.Minute),
	}))
	require.NoError(t, models.DeleteMCPOAuthRefreshTokensForClient(db, claims.OrgID, claims.FactoryID, claims.UserID, claims.ClientID))
	require.NoError(t, models.DeleteMCPOAuthCodesForClient(db, claims.OrgID, claims.FactoryID, claims.UserID, claims.ClientID))

	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", code)
	values.Set("code_verifier", mcpTestPKCEVerifier)
	values.Set("client_id", claims.ClientID)
	values.Set("redirect_uri", mcpserver.CursorRedirectURIs[0])
	values.Set("resource", claims.Resource)
	tokenReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(values.Encode()))
	tokenReq.Host = "localhost:8000"
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRec := httptest.NewRecorder()
	server.Router.ServeHTTP(tokenRec, tokenReq)
	assert.Equal(t, http.StatusBadRequest, tokenRec.Code)
	assert.Contains(t, tokenRec.Body.String(), "invalid_grant")

	req := mcpRequest(http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMCPProtectedResourceMetadataOnResourcePath(t *testing.T) {
	_, server, _ := mcpEnabledServer(t)
	req := mcpRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", "")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "http://localhost:8000/mcp", body["resource"])
}

func TestMCPAuthorizationServerMetadataOnResourcePath(t *testing.T) {
	_, server, _ := mcpEnabledServer(t)
	req := mcpRequest(http.MethodGet, "/.well-known/oauth-authorization-server/mcp", "")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "http://localhost:8000", body["issuer"])
	assert.Equal(t, "http://localhost:8000/oauth/authorize", body["authorization_endpoint"])
	assert.Equal(t, "http://localhost:8000/oauth/register", body["registration_endpoint"])
}

func TestMCPUnauthenticatedGETReturns401(t *testing.T) {
	_, server, _ := mcpEnabledServer(t)
	req := mcpRequest(http.MethodGet, "/mcp", "")
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "http://localhost:8000/.well-known/oauth-protected-resource/mcp")
}
