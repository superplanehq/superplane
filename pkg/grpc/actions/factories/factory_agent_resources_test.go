package factories

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func enableWorkspaceMCPAndSkills(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureWorkspaceMCP))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureWorkspaceSkills))
}

func Test__DeleteFactoryAgentResourceRevokesOAuth(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	var revoked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), "refresh-token")
		revoked.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://api.mobbin.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		ClientID:           "client-1",
		RevocationEndpoint: server.URL,
	}))
	encrypted, err := mcp.EncryptResourceSecret(t.Context(), r.Encryptor, resource.ID, "refresh-token")
	require.NoError(t, err)
	require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, encrypted))

	_, err = DeleteFactoryAgentResource(t.Context(), IntakeDependencies{Encryptor: r.Encryptor}, r.Organization.ID.String(), &pb.DeleteFactoryAgentResourceRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
	})
	require.NoError(t, err)
	assert.True(t, revoked.Load())
	_, err = factory.FindAgentResource(db, resource.ID)
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceNotFound)
}

func Test__UpdateFactoryAgentResourceRevokesOAuthOnURLChange(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	var revoked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		revoked.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://api.mobbin.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		ClientID:           "client-1",
		RevocationEndpoint: server.URL,
	}))
	encrypted, err := mcp.EncryptResourceSecret(t.Context(), r.Encryptor, resource.ID, "refresh-token")
	require.NoError(t, err)
	require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, encrypted))

	nextURL := "https://other.example/mcp"
	response, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{Encryptor: r.Encryptor}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
		Url:        &nextURL,
	})
	require.NoError(t, err)
	assert.True(t, revoked.Load())
	assert.Equal(t, nextURL, response.GetResource().GetUrl())
	assert.Equal(t, pb.FactoryAgentResource_OAUTH_STATUS_NOT_CONNECTED, response.GetResource().GetOauthStatus())
	_, err = resource.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceSecretNotFound)
}

func Test__UpdateFactoryAgentResourceKeepsOAuthWhenUpdateFails(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "taken", false, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	})
	require.NoError(t, err)

	var revoked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		revoked.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://api.mobbin.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		ClientID:           "client-1",
		RevocationEndpoint: server.URL,
	}))
	encrypted, err := mcp.EncryptResourceSecret(t.Context(), r.Encryptor, resource.ID, "refresh-token")
	require.NoError(t, err)
	require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, encrypted))

	taken := "taken"
	nextURL := "https://other.example/mcp"
	_, err = UpdateFactoryAgentResource(t.Context(), IntakeDependencies{Encryptor: r.Encryptor}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{
		FactoryId:  factory.ID.String(),
		ResourceId: resource.ID.String(),
		Name:       &taken,
		Url:        &nextURL,
	})
	require.Error(t, err)
	assert.False(t, revoked.Load())

	reloaded, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	assert.Equal(t, "mobbin", reloaded.Name)
	assert.Equal(t, "https://api.mobbin.com/mcp", reloaded.Config.Data().URL)
	assert.Equal(t, models.FactoryAgentResourceOAuthConnected, reloaded.OAuthState())
	secret, err := resource.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Value)
}

func Test__CreateFactoryAgentResourceCreatesInlineSkill(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	response, err := CreateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_SKILL,
		Name:      "review-copy",
		Enabled:   true,
		Markdown:  "# Review copy\n\nWrite STE UI copy.",
	})
	require.NoError(t, err)
	require.NotNil(t, response.GetResource())
	assert.Equal(t, pb.FactoryAgentResource_KIND_SKILL, response.GetResource().GetKind())
	assert.Equal(t, "review-copy", response.GetResource().GetName())
	assert.Equal(t, "# Review copy\n\nWrite STE UI copy.", response.GetResource().GetMarkdown())
}

func Test__CreateFactoryAgentResourceRejectsEmptySkillMarkdown(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = CreateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_SKILL,
		Name:      "ui-ux",
		Enabled:   true,
	})
	require.Error(t, err)
}

func Test__CreateFactoryAgentResourceRejectsConnectedURL(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = CreateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_MCP_SERVER,
		Name:      "docs",
		Enabled:   true,
		Url:       "https://mcp.example.com/mcp",
		Auth:      pb.FactoryAgentResource_AUTH_HEADERS,
		Headers: []*pb.FactoryAgentResource_Header{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	})
	require.NoError(t, err)

	_, err = CreateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_MCP_SERVER,
		Name:      "docs-copy",
		Enabled:   true,
		Url:       "https://mcp.example.com/mcp/",
		Auth:      pb.FactoryAgentResource_AUTH_HEADERS,
		Headers: []*pb.FactoryAgentResource_Header{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	assert.Equal(t, "This MCP server is already connected.", grpcerrors.StatusMessage(err))
}

func Test__UpdateFactoryAgentResourceStoresDisabledTools(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)
	replaceDisabled := true
	response, err := UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{
		FactoryId:            factory.ID.String(),
		ResourceId:           resource.ID.String(),
		DisabledTools:        []string{"create_issue", " search "},
		ReplaceDisabledTools: &replaceDisabled,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"create_issue", "search"}, response.GetResource().GetDisabledTools())

	saved, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	assert.True(t, saved.Config.Data().ToolsDefaultApplied)
}

func Test__UpdateFactoryAgentResourceEmptyToolListSetsDefaultApplied(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)
	replaceDisabled := true
	_, err = UpdateFactoryAgentResource(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryAgentResourceRequest{
		FactoryId:            factory.ID.String(),
		ResourceId:           resource.ID.String(),
		ReplaceDisabledTools: &replaceDisabled,
	})
	require.NoError(t, err)

	saved, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	assert.True(t, saved.Config.Data().ToolsDefaultApplied)
	assert.Empty(t, saved.Config.Data().DisabledTools)
}

func Test__CreateFactoryAgentResourceDefaultsWriteToolsOff(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	server := httptest.NewServer(mcpToolsHandler(t, []map[string]any{
		{"name": "search", "annotations": map[string]any{"readOnlyHint": true}},
		{"name": "create_issue"},
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       server.URL,
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)
	applyDefaultMCPWriteTools(t.Context(), IntakeDependencies{}, db, resource)

	saved, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	assert.True(t, saved.Config.Data().ToolsDefaultApplied)
	assert.Equal(t, []string{"create_issue"}, saved.Config.Data().DisabledTools)
}

func Test__CreateFactoryAgentResourceKeepsGoingWhenToolListFails(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	response, err := CreateFactoryAgentResource(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_MCP_SERVER,
		Name:      "docs",
		Enabled:   true,
		Url:       "https://192.0.2.1/mcp",
		Auth:      pb.FactoryAgentResource_AUTH_HEADERS,
	})
	require.NoError(t, err)

	saved, err := factory.FindAgentResource(db, uuid.MustParse(response.GetResource().GetId()))
	require.NoError(t, err)
	assert.False(t, saved.Config.Data().ToolsDefaultApplied)
	assert.Empty(t, saved.Config.Data().DisabledTools)
}

func Test__finishFactoryAgentResourceOAuthConnectLeavesDefaultUnsetWhenListFails(t *testing.T) {
	r := support.Setup(t)
	enableWorkspaceMCPAndSkills(t, r.Organization.ID)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"access","token_type":"Bearer","expires_in":3600}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "<html>not json</html>")
	}))
	t.Cleanup(server.Close)

	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       server.URL,
		Auth:      models.FactoryAgentResourceAuthOAuth,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		TokenEndpoint: server.URL + "/token",
		ClientID:      "client-1",
	}))
	encrypted, err := mcp.EncryptResourceSecret(t.Context(), r.Encryptor, resource.ID, "refresh-token")
	require.NoError(t, err)
	require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, encrypted))

	finishFactoryAgentResourceOAuthConnect(t.Context(), IntakeDependencies{Encryptor: r.Encryptor}, db, resource)

	saved, err := factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	assert.False(t, saved.Config.Data().ToolsDefaultApplied)
	assert.Empty(t, saved.Config.Data().DisabledTools)
}

func Test__finishFactoryAgentResourceOAuthConnectRejectsDuplicateURL(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	oauth := models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://api.mobbin.com/mcp",
		Auth:      models.FactoryAgentResourceAuthOAuth,
	}
	first, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, oauth)
	require.NoError(t, err)
	second, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin-retry", true, oauth)
	require.NoError(t, err)
	require.NoError(t, first.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	require.NoError(t, second.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, []byte("refresh")))
	require.NoError(t, second.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, []byte("access")))

	finishFactoryAgentResourceOAuthConnect(t.Context(), IntakeDependencies{}, db, second)

	assert.Equal(t, models.FactoryAgentResourceOAuthNeedsReconnect, second.OAuthState())
	assert.Equal(t, "This MCP server is already connected.", second.OAuthError)
	_, err = second.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceSecretNotFound)
	_, err = second.FindSecret(db, models.FactoryAgentResourceSecretAccessToken)
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceSecretNotFound)
}
