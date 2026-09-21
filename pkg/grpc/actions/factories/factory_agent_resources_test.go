package factories

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__DeleteFactoryAgentResourceRevokesOAuth(t *testing.T) {
	r := support.Setup(t)
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
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	response, err := CreateFactoryAgentResource(t.Context(), r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
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
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = CreateFactoryAgentResource(t.Context(), r.Organization.ID.String(), &pb.CreateFactoryAgentResourceRequest{
		FactoryId: factory.ID.String(),
		Kind:      pb.FactoryAgentResource_KIND_SKILL,
		Name:      "ui-ux",
		Enabled:   true,
	})
	require.Error(t, err)
}
