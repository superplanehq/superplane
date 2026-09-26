package models_test

import (
	"fmt"
	"strings"
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
		other := headerConfig
		other.URL = "https://mcp.other.example/mcp"
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", false, other)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceNameTaken)
	})

	t.Run("creates an inline skill", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "Review-Copy", true, models.FactoryAgentResourceConfig{
			Source:   models.FactoryAgentResourceSourceInline,
			Markdown: "# Review copy\n\nWrite STE UI copy.",
		})
		require.NoError(t, err)
		assert.Equal(t, "review-copy", resource.Name)
		assert.Equal(t, models.FactoryAgentResourceKindSkill, resource.Kind)
		assert.Equal(t, models.FactoryAgentResourceSourceInline, resource.Config.Data().Source)
		assert.Equal(t, "# Review copy\n\nWrite STE UI copy.", resource.Config.Data().Markdown)

		enabled, err := factory.ListEnabledSkills(db)
		require.NoError(t, err)
		require.Len(t, enabled, 1)
		assert.Equal(t, resource.ID, enabled[0].ID)

		found, err := factory.FindAgentResource(db, resource.ID)
		require.NoError(t, err)
		assert.Equal(t, resource.ID, found.ID)
	})

	t.Run("rejects GitHub skill packages", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "ui-ux", true, models.FactoryAgentResourceConfig{
			Source:     "github",
			Repository: "nextlevelbuilder/ui-ux-pro-max-skill",
			Ref:        "main",
		})
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceKindNotSupported)
	})

	t.Run("rejects empty skill markdown", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "empty", true, models.FactoryAgentResourceConfig{
			Source: models.FactoryAgentResourceSourceInline,
		})
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceMarkdownRequired)
	})

	t.Run("rejects oversized skill markdown", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "huge", true, models.FactoryAgentResourceConfig{
			Source:   models.FactoryAgentResourceSourceInline,
			Markdown: strings.Repeat("a", models.MaxFactoryAgentSkillMarkdownBytes+1),
		})
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceMarkdownTooLarge)
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
		off := headerConfig
		off.URL = "https://mcp.example.com/mcp-off"
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "off", false, off)
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
			cfg := headerConfig
			cfg.URL = fmt.Sprintf("https://mcp.example.com/mcp/%02d", i)
			_, err = factory.CreateAgentResource(
				db,
				models.FactoryAgentResourceKindMCPServer,
				fmt.Sprintf("mcp-%02d", i),
				true,
				cfg,
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

	t.Run("rejects a second connected MCP at the same URL", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, headerConfig)
		require.NoError(t, err)

		duplicate := headerConfig
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs-copy", true, duplicate)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLTaken)

		slash := headerConfig
		slash.URL = "https://mcp.example.com/mcp/"
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs-slash", false, slash)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLTaken)
	})

	t.Run("rejects a root URL with or without a trailing slash", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		root := headerConfig
		root.URL = "https://mcp.example.com"
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "root", true, root)
		require.NoError(t, err)
		slash := root
		slash.URL = "https://mcp.example.com/"
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "root-slash", true, slash)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLTaken)
	})

	t.Run("allows a second OAuth server that is not connected yet", func(t *testing.T) {
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
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, first.OAuthState())
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, second.OAuthState())

		require.NoError(t, first.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
		err = second.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLTaken)
		assert.Equal(t, models.FactoryAgentResourceOAuthNotConnected, second.OAuthState())
	})

	t.Run("allows an update that keeps the same connected URL", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, headerConfig)
		require.NoError(t, err)
		same := resource.Config.Data()
		require.NoError(t, resource.Update(db, nil, nil, &same))
		assert.Equal(t, headerConfig.URL, resource.Config.Data().URL)
	})

	t.Run("rejects an update that reuses another connected URL", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, headerConfig)
		require.NoError(t, err)
		other := headerConfig
		other.URL = "https://mcp.other.example/mcp"
		resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "other", true, other)
		require.NoError(t, err)
		collide := other
		collide.URL = headerConfig.URL
		err = resource.Update(db, nil, nil, &collide)
		assert.ErrorIs(t, err, models.ErrFactoryAgentResourceURLTaken)
		reloaded, err := factory.FindAgentResource(db, resource.ID)
		require.NoError(t, err)
		assert.Equal(t, other.URL, reloaded.Config.Data().URL)
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
	assert.NoError(t, err)

	err = models.FactoryAgentResourceConfig{
		URL:  "https://mcp.example.com/mcp",
		Auth: models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name: "Authorization",
		}},
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

func Test__CanonicalMCPServerURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, models.CanonicalMCPServerURL("https://mcp.sentry.dev/mcp"), models.CanonicalMCPServerURL("https://mcp.sentry.dev/mcp/"))
	assert.Equal(t, models.CanonicalMCPServerURL("https://mcp.sentry.dev/mcp"), models.CanonicalMCPServerURL("https://MCP.Sentry.DEV/mcp"))
	assert.Equal(t, models.CanonicalMCPServerURL("https://mcp.example.com"), models.CanonicalMCPServerURL("https://mcp.example.com/"))
	assert.Empty(t, models.CanonicalMCPServerURL("  "))
}
