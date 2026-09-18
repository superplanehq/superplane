package runner_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAttachWorkspaceAgentResourcesSkipsNonFactoryCanvas(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	assert.Empty(t, environment)
	assert.Empty(t, files)
}

func TestAttachWorkspaceAgentResourcesWritesHeaderServers(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("NO_ENCRYPTION", "yes")
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceAgentResources))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
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
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "off", false, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/other",
		Auth:      models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"vendor-mcp/token": []byte("secret-token"),
			},
		},
	}, nil, nil)
	require.Len(t, files, 1)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[0].Path)
	require.Len(t, environment, 1)
	assert.Equal(t, runner.EnvSuperplaneWorkspaceMCPConfig, environment[0].Name)

	var payload struct {
		Servers []struct {
			Name    string            `json:"name"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[0].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "docs", payload.Servers[0].Name)
	assert.Equal(t, "https://mcp.example.com/mcp", payload.Servers[0].URL)
	assert.Equal(t, "secret-token", payload.Servers[0].Headers["Authorization"])
}

func TestAttachWorkspaceAgentResourcesWritesPublicServersWithoutHeaders(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceAgentResources))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.deepwiki.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	require.Len(t, files, 1)
	require.Len(t, environment, 1)
	var payload struct {
		Servers []struct {
			Name    string            `json:"name"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[0].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "deepwiki", payload.Servers[0].Name)
	assert.Equal(t, "https://mcp.deepwiki.com/mcp", payload.Servers[0].URL)
	assert.Empty(t, payload.Servers[0].Headers)
}

func TestAttachWorkspaceAgentResourcesKeepsServerWhenHeaderSecretMissing(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceAgentResources))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.deepwiki.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "X-Unused",
			SecretName: "missing-secret",
			SecretKey:  "token",
		}},
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
	}, nil, nil)
	require.Len(t, files, 1)
	require.Len(t, environment, 1)
	var payload struct {
		Servers []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[0].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "deepwiki", payload.Servers[0].Name)
	assert.Equal(t, "https://mcp.deepwiki.com/mcp", payload.Servers[0].URL)
}

func TestAttachWorkspaceAgentResourcesSkipsWhenFeatureDisabled(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
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

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Secrets: &contexts.SecretsContext{
			Values: map[string][]byte{
				"vendor-mcp/token": []byte("secret-token"),
			},
		},
	}, nil, nil)
	assert.Empty(t, environment)
	assert.Empty(t, files)
}
