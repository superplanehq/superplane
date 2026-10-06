package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func TestMCPAPITokenBearerCallsToolsWithoutOAuthGrant(t *testing.T) {
	r, server, _ := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "OpenCode", "", "OPC")
	require.NoError(t, err)
	resource := "http://localhost:8000/mcp"
	created, err := factoryactions.CreateFactoryMCPAPIToken(mcpTokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  resource,
	})
	require.NoError(t, err)
	secret := created.GetPlaintext()

	initialize := postMCP(t, server, secret, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	require.Equal(t, http.StatusOK, initialize.Code)
	assert.Contains(t, initialize.Body.String(), "protocolVersion")
	assert.NotContains(t, initialize.Body.String(), `"error"`)

	toolCall := postMCP(t, server, secret, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_tasks","arguments":{}}}`)
	require.Equal(t, http.StatusOK, toolCall.Code)
	assert.NotContains(t, toolCall.Body.String(), `"error"`)

	stored, err := models.FindMCPAPITokenByHash(db, crypto.HashToken(secret))
	require.NoError(t, err)
	require.NotNil(t, stored.LastUsedAt)
	assert.WithinDuration(t, time.Now(), *stored.LastUsedAt, 5*time.Second)

	var grants int64
	require.NoError(t, db.Model(&models.MCPOAuthRefreshToken{}).
		Where("user_id = ? AND factory_id = ?", r.User, factory.ID).
		Count(&grants).Error)
	assert.Zero(t, grants)
}

func TestMCPAPITokenRevokeAndMismatchReturn401(t *testing.T) {
	r, server, _ := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, "OpenCode", "", "OPC")
	require.NoError(t, err)
	created, err := factoryactions.CreateFactoryMCPAPIToken(mcpTokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  "https://other.example/mcp",
	})
	require.NoError(t, err)

	mismatch := postMCP(t, server, created.GetPlaintext(), `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	assert.Equal(t, http.StatusUnauthorized, mismatch.Code)
	assert.Contains(t, mismatch.Header().Get("WWW-Authenticate"), "oauth-protected-resource")

	matching, err := factoryactions.CreateFactoryMCPAPIToken(mcpTokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  "http://localhost:8000/mcp",
	})
	require.NoError(t, err)
	_, err = factoryactions.RevokeFactoryMCPAPIToken(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		TokenId:   matching.GetToken().GetId(),
	})
	require.NoError(t, err)
	revoked := postMCP(t, server, matching.GetPlaintext(), `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	assert.Equal(t, http.StatusUnauthorized, revoked.Code)
}

func TestMCPRejectsPersonalTokenAndOrganizationAPIKey(t *testing.T) {
	r, server, _ := mcpEnabledServer(t)
	personal, err := crypto.Base64String(32)
	require.NoError(t, err)
	require.NoError(t, models.CreateUserAPIToken(database.DB(t.Context()), models.NewUserAPIToken(r.User, "CI", crypto.HashToken(personal))))

	apiKeyRaw, err := crypto.Base64String(32)
	require.NoError(t, err)
	apiKey, err := models.CreateAPIKey(database.DB(t.Context()), r.Organization.ID, "org-key", nil, r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, apiKey.UpdateTokenHash(crypto.HashToken(apiKeyRaw)))

	for _, bearer := range []string{personal, apiKeyRaw} {
		rec := postMCP(t, server, bearer, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "oauth-protected-resource")
	}
}

func TestMCPAPITokenBlockedAccountReturns401(t *testing.T) {
	r, server, _ := mcpEnabledServer(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, "OpenCode", "", "OPC")
	require.NoError(t, err)
	created, err := factoryactions.CreateFactoryMCPAPIToken(mcpTokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  "http://localhost:8000/mcp",
	})
	require.NoError(t, err)
	require.NoError(t, r.Account.Block(database.DB(t.Context()), time.Now()))

	rec := postMCP(t, server, created.GetPlaintext(), `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func postMCP(t *testing.T, server *Server, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := mcpRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func mcpTokenContext(r *support.ResourceRegistry) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-user-id", r.User.String(),
	))
}
