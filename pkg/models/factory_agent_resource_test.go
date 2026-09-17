package models_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryAgentResource(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	headerConfig := models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	}

	t.Run("creates a header MCP connection", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "Linear-Docs", true, headerConfig)
		require.NoError(t, err)
		assert.Equal(t, "linear-docs", resource.Name)
		assert.True(t, resource.Enabled)
		assert.Equal(t, "https://mcp.example.com/mcp", resource.Config.Data().URL)

		found, err := factory.FindAgentResource(db, resource.ID)
		require.NoError(t, err)
		assert.Equal(t, resource.ID, found.ID)
	})

	t.Run("rejects reserved name", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "superplane", true, headerConfig)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceNameReserved)
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, headerConfig)
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", false, headerConfig)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceNameTaken)
	})

	t.Run("rejects skill kind in v1", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "ui-ux", true, models.FactoryAgentResourceConfig{
			Source:     "github",
			Repository: "nextlevelbuilder/ui-ux-pro-max-skill",
			Ref:        "main",
		})
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceKindNotSupported)
	})

	t.Run("oauth row starts not connected", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin", true, models.FactoryAgentResourceConfig{
			Transport: "http",
			URL:       "https://api.mobbin.com/mcp",
			Auth:      models.FactoryAgentResourceAuthOAuth,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, resource.OAuthState())
	})

	t.Run("lists enabled MCP servers only", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "on", true, headerConfig)
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "off", false, headerConfig)
		require.NoError(t, err)

		enabled, err := factory.ListEnabledMCPServers(db)
		require.NoError(t, err)
		require.Len(t, enabled, 1)
		assert.Equal(t, "on", enabled[0].Name)
	})

	t.Run("stores encrypted secret bytes", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "oauth-secret", true, models.FactoryAgentResourceConfig{
			URL:  "https://api.mobbin.com/mcp",
			Auth: models.FactoryAgentResourceAuthOAuth,
		})
		require.NoError(t, err)
		require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, []byte("refresh")))
		secret, err := resource.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
		require.NoError(t, err)
		assert.Equal(t, []byte("refresh"), secret.Value)
	})

	t.Run("deletes secrets with the resource", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "to-delete", true, headerConfig)
		require.NoError(t, err)
		require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretAccessToken, []byte("access")))
		require.NoError(t, resource.Delete(db))
		_, err = factory.FindAgentResource(db, resource.ID)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceNotFound)
	})

	t.Run("finds factory id for a factory canvas", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
		id, err := models.FindFactoryIDForCanvas(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, id)
		assert.Equal(t, factory.ID, *id)
	})

	t.Run("returns nil factory id for a non-factory canvas", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		id, err := models.FindFactoryIDForCanvas(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Nil(t, id)
	})

	t.Run("resets oauth when the MCP URL changes", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin-url", true, models.FactoryAgentResourceConfig{
			Transport: "http",
			URL:       "https://api.mobbin.com/mcp",
			Auth:      models.FactoryAgentResourceAuthOAuth,
		})
		require.NoError(t, err)
		require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
		require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
			ClientID:           "old-client",
			RevocationEndpoint: "https://auth.example/revoke",
		}))
		require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, []byte("refresh")))

		next := models.FactoryAgentResourceConfig{
			Transport: "http",
			URL:       "https://other.example/mcp",
			Auth:      models.FactoryAgentResourceAuthOAuth,
		}
		require.NoError(t, resource.Update(db, nil, nil, &next))
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, resource.OAuthState())
		assert.Empty(t, resource.OAuthMetadata.Data().ClientID)
		_, err = resource.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceSecretNotFound)

		reloaded, err := factory.FindAgentResource(db, resource.ID)
		require.NoError(t, err)
		assert.Equal(t, "https://other.example/mcp", reloaded.Config.Data().URL)
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, reloaded.OAuthState())
		assert.Empty(t, reloaded.OAuthMetadata.Data().ClientID)
	})

	t.Run("keeps oauth when the URL does not change", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mobbin-keep", true, models.FactoryAgentResourceConfig{
			Transport: "http",
			URL:       "https://api.mobbin.com/mcp",
			Auth:      models.FactoryAgentResourceAuthOAuth,
		})
		require.NoError(t, err)
		require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
		require.NoError(t, resource.UpsertSecret(db, models.FactoryAgentResourceSecretRefreshToken, []byte("refresh")))

		same := resource.Config.Data()
		require.NoError(t, resource.Update(db, nil, nil, &same))
		assert.Equal(t, models.FactoryAgentResourceOAuthConnected, resource.OAuthState())
		secret, err := resource.FindSecret(db, models.FactoryAgentResourceSecretRefreshToken)
		require.NoError(t, err)
		assert.Equal(t, []byte("refresh"), secret.Value)
	})

	t.Run("rejects a 21st enabled MCP connection", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		for i := range models.MaxEnabledFactoryMCPServers {
			_, err = factory.CreateAgentResource(
				db,
				models.FactoryAgentResourceKindMCPServer,
				fmt.Sprintf("mcp-%02d", i),
				true,
				headerConfig,
			)
			require.NoError(t, err)
		}
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mcp-over", true, headerConfig)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceMCPCapReached)

		extra, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "mcp-over", false, headerConfig)
		require.NoError(t, err)
		enabled := true
		err = extra.Update(db, nil, &enabled, nil)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceMCPCapReached)
	})
}

func Test__ValidateFactoryAgentResourceName(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, models.ValidateFactoryAgentResourceName(""), models.ErrFactoryAgentResourceNameInvalid)
	assert.ErrorIs(t, models.ValidateFactoryAgentResourceName("Superplane"), models.ErrFactoryAgentResourceNameInvalid)
	assert.ErrorIs(t, models.ValidateFactoryAgentResourceName("superplane"), models.ErrFactoryAgentResourceNameReserved)
	assert.NoError(t, models.ValidateFactoryAgentResourceName("linear-docs"))
}

func Test__FactoryAgentResourceConfigValidateMCP(t *testing.T) {
	t.Parallel()
	err := models.FactoryAgentResourceConfig{Auth: models.FactoryAgentResourceAuthOAuth}.ValidateMCP()
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLRequired)

	err = models.FactoryAgentResourceConfig{
		URL:  "https://mcp.example.com/mcp",
		Auth: models.FactoryAgentResourceAuthHeaders,
	}.ValidateMCP()
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceHeaderInvalid)

	err = models.FactoryAgentResourceConfig{
		URL:  "https://mcp.example.com/mcp",
		Auth: "basic",
	}.ValidateMCP()
	assert.ErrorIs(t, err, models.ErrFactoryAgentResourceAuthInvalid)

	oauth := models.FactoryAgentResourceConfig{
		URL:  "https://api.mobbin.com/mcp",
		Auth: models.FactoryAgentResourceAuthOAuth,
	}
	assert.True(t, oauth.InvalidatesOAuth(models.FactoryAgentResourceConfig{
		URL:  "https://other.example/mcp",
		Auth: models.FactoryAgentResourceAuthOAuth,
	}))
	assert.True(t, oauth.InvalidatesOAuth(models.FactoryAgentResourceConfig{
		URL:  "https://api.mobbin.com/mcp",
		Auth: models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	}))
	assert.False(t, oauth.InvalidatesOAuth(oauth))

	_ = uuid.Nil
}
