package runner_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/gorm"
)

func TestAttachWorkspaceAgentResourcesRefreshesOAuthTokenThatWillNotLastTheRun(t *testing.T) {
	t.Run("long run refreshes a short token", func(t *testing.T) {
		header, calls := attachOAuthWorkspaceMCP(t, 3*time.Minute, 3600)
		assert.Equal(t, "Bearer fresh-access", header)
		assert.Equal(t, 1, calls)
	})

	t.Run("short run reuses a token that outlasts it", func(t *testing.T) {
		header, calls := attachOAuthWorkspaceMCP(t, 3*time.Minute, 60)
		assert.Equal(t, "Bearer cached-access", header)
		assert.Zero(t, calls)
	})
}

func setupWorkspaceAgentResources(t *testing.T) *support.ResourceRegistry {
	t.Helper()
	r := support.Setup(t)
	crypto.SetProcessEncryptor(r.Encryptor)
	t.Cleanup(func() { crypto.SetProcessEncryptor(nil) })
	return r
}

func attachOAuthWorkspaceMCP(t *testing.T, expiresIn time.Duration, timeoutSeconds int) (string, int) {
	t.Helper()
	r := setupWorkspaceAgentResources(t)
	t.Setenv("NO_ENCRYPTION", "yes")
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "sentry", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthOAuth,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	require.NoError(t, resource.SetOAuthMetadata(db, models.FactoryAgentResourceOAuthMetadata{
		TokenEndpoint:        "https://example.com/oauth/token",
		ClientID:             "client-1",
		AccessTokenExpiresAt: time.Now().Add(expiresIn).UTC().Format(time.RFC3339),
	}))
	require.NoError(t, resource.SetOAuthStatus(db, models.FactoryAgentResourceOAuthConnected, "", nil))
	encryptor := r.Encryptor
	require.NoError(t, storeWorkspaceOAuthSecret(t, encryptor, db, resource, models.FactoryAgentResourceSecretAccessToken, "cached-access"))
	require.NoError(t, storeWorkspaceOAuthSecret(t, encryptor, db, resource, models.FactoryAgentResourceSecretRefreshToken, "refresh-1"))

	httpClient := &countingTokenHTTP{}
	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		HTTP:           httpClient,
		Configuration:  map[string]any{"executionTimeoutSeconds": timeoutSeconds},
	}, nil, nil)
	require.NotEmpty(t, files)
	var raw string
	for _, file := range files {
		if file.Path == runner.WorkspaceMCPConfigPath {
			raw = file.Content
			break
		}
	}
	require.NotEmpty(t, raw)

	var payload struct {
		Servers []struct {
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &payload))
	require.Len(t, payload.Servers, 1)
	return payload.Servers[0].Headers["Authorization"], httpClient.calls
}

func storeWorkspaceOAuthSecret(t *testing.T, encryptor crypto.Encryptor, db *gorm.DB, resource *models.FactoryAgentResource, name, value string) error {
	t.Helper()
	encrypted, err := mcp.EncryptResourceSecret(t.Context(), encryptor, resource.ID, value)
	if err != nil {
		return err
	}
	return resource.UpsertSecret(db, name, encrypted)
}

type countingTokenHTTP struct {
	calls int
}

func (c *countingTokenHTTP) Do(req *http.Request) (*http.Response, error) {
	c.calls++
	body := `{"access_token":"fresh-access","refresh_token":"refresh-2","expires_in":3600,"token_type":"Bearer"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestAttachWorkspaceAgentResourcesSkipsNonFactoryCanvas(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	assert.Empty(t, environment)
	assert.Empty(t, files)
}

func TestAttachWorkspaceAgentResourcesWritesHeaderServers(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
			SecretName: "vendor-mcp",
			SecretKey:  "token",
		}},
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "off", false, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/other",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
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
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[1].Path)
	require.Len(t, environment, 1)
	assert.Equal(t, runner.EnvSuperplaneWorkspaceMCPConfig, environment[0].Name)
	assert.Equal(t, "$SUPERPLANE_TASK_DIR/"+runner.WorkspaceMCPConfigPath, environment[0].Value)

	var payload struct {
		Servers []struct {
			Name    string            `json:"name"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[1].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "docs", payload.Servers[0].Name)
	assert.Equal(t, "https://mcp.example.com/mcp", payload.Servers[0].URL)
	assert.Equal(t, "secret-token", payload.Servers[0].Headers["Authorization"])
}

func TestAttachWorkspaceAgentResourcesWritesPublicServersWithoutHeaders(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.deepwiki.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	require.Len(t, environment, 1)
	var payload struct {
		Servers []struct {
			Name    string            `json:"name"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[1].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "deepwiki", payload.Servers[0].Name)
	assert.Equal(t, "https://mcp.deepwiki.com/mcp", payload.Servers[0].URL)
	assert.Empty(t, payload.Servers[0].Headers)
}

func TestAttachWorkspaceAgentResourcesSkipsServerUntilDefaultsAreSaved(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport: "http",
		URL:       "https://mcp.example.com/mcp",
		Auth:      models.FactoryAgentResourceAuthHeaders,
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	assert.Empty(t, environment)
	require.Len(t, files, 1)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
}

func TestAttachWorkspaceAgentResourcesSkipsServerWhenHeaderSecretMissing(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
		Headers: []models.FactoryAgentResourceHeader{{
			Name:       "Authorization",
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
	assert.Empty(t, environment)
	require.Len(t, files, 1)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
}

func TestAttachWorkspaceAgentResourcesAttachesMCPWithoutFlag(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
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
	require.Len(t, environment, 1)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[1].Path)
}

func TestAttachWorkspaceAgentResourcesWritesInlineSkills(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceSkills))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy\n\nWrite STE UI copy.",
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "off", false, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Disabled",
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, []runner.TaskFile{
		{Path: "prompts/refine.txt", Content: "Refine the request.\n", Mode: "0644"},
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n\n", Mode: "0644"},
	})
	assert.Empty(t, environment)
	require.Len(t, files, 5)
	assert.Equal(t, "Refine the request.\n", files[0].Content)
	assert.Equal(t, "Implement the plan.\n\n", files[1].Content)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[2].Path)
	assert.Equal(t, ".claude/skills/review-copy/SKILL.md", files[3].Path)
	assert.Equal(t, ".agents/skills/review-copy/SKILL.md", files[4].Path)
	assert.Equal(t, "# Review copy\n\nWrite STE UI copy.\n", files[3].Content)
	assert.Equal(t, files[3].Content, files[4].Content)
}

func TestAttachWorkspaceAgentResourcesWritesMCPAndSkills(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceSkills))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.deepwiki.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy",
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	require.Len(t, environment, 1)
	require.Len(t, files, 4)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[1].Path)
	assert.Equal(t, ".claude/skills/review-copy/SKILL.md", files[2].Path)
	assert.Equal(t, ".agents/skills/review-copy/SKILL.md", files[3].Path)
}

func TestAttachWorkspaceAgentResourcesSkipsDisabledIDsAndHintsPrompts(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceSkills))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	kept, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.deepwiki.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	disabled, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "linear", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.linear.app/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy",
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"disabledAgentResourceIds": []any{disabled.ID.String()},
		},
	}, nil, []runner.TaskFile{
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n", Mode: "0644"},
	})
	require.Len(t, environment, 1)
	require.Len(t, files, 5)
	assert.Equal(t, "prompts/implement.txt", files[0].Path)
	assert.Contains(t, files[0].Content, "You can use these resources.")
	assert.Contains(t, files[0].Content, "MCP servers: "+kept.Name)
	assert.NotContains(t, files[0].Content, "Skills:")
	assert.NotContains(t, files[0].Content, "review-copy")
	assert.NotContains(t, files[0].Content, "linear")
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[1].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[2].Path)
	var payload struct {
		Servers []struct {
			Name string `json:"name"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[2].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "deepwiki", payload.Servers[0].Name)
}

func TestAttachWorkspaceAgentResourcesMergesDisabledTools(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
		DisabledTools:       []string{"search"},
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy",
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"disabledAgentResourceTools": map[string]any{
				resource.ID.String(): []any{"create_issue"},
			},
		},
	}, nil, nil)
	require.Len(t, environment, 1)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[1].Path)
	var payload struct {
		Servers []struct {
			Name          string   `json:"name"`
			DisabledTools []string `json:"disabledTools"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[1].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, "docs", payload.Servers[0].Name)
	assert.Equal(t, []string{"search", "create_issue"}, payload.Servers[0].DisabledTools)
}

func TestAttachWorkspaceAgentResourcesOmitsEnabledOverride(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
		DisabledTools:       []string{"create_issue", "search"},
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"enabledAgentResourceTools": map[string]any{
				resource.ID.String(): []any{"create_issue"},
			},
			"disabledAgentResourceTools": map[string]any{
				resource.ID.String(): []any{"update_issue"},
			},
		},
	}, nil, nil)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	var payload struct {
		Servers []struct {
			DisabledTools []string `json:"disabledTools"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[1].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, []string{"search", "update_issue"}, payload.Servers[0].DisabledTools)
}

func TestAttachWorkspaceAgentResourcesKeepsToolOffWhenBothListsNameIt(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	resource, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
		DisabledTools:       []string{"create_issue"},
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"enabledAgentResourceTools": map[string]any{
				resource.ID.String(): []any{"create_issue"},
			},
			"disabledAgentResourceTools": map[string]any{
				resource.ID.String(): []any{"create_issue"},
			},
		},
	}, nil, nil)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	var payload struct {
		Servers []struct {
			DisabledTools []string `json:"disabledTools"`
		} `json:"servers"`
	}
	require.NoError(t, json.Unmarshal([]byte(files[1].Content), &payload))
	require.Len(t, payload.Servers, 1)
	assert.Equal(t, []string{"create_issue"}, payload.Servers[0].DisabledTools)
}

func TestAttachWorkspaceAgentResourcesAttachesMCPOnlyWhenSkillsFlagIsOff(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "docs", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.example.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "review-copy", true, models.FactoryAgentResourceConfig{
		Source:   models.FactoryAgentResourceSourceInline,
		Markdown: "# Review copy",
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	require.Len(t, environment, 1)
	require.Len(t, files, 2)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, runner.WorkspaceMCPConfigPath, files[1].Path)
}

func TestAttachWorkspaceAgentResourcesOmitsHintWhenAllDisabled(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	disabled, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindMCPServer, "deepwiki", true, models.FactoryAgentResourceConfig{
		Transport:           "http",
		URL:                 "https://mcp.deepwiki.com/mcp",
		Auth:                models.FactoryAgentResourceAuthHeaders,
		ToolsDefaultApplied: true,
	})
	require.NoError(t, err)

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"disabledAgentResourceIds": []string{disabled.ID.String()},
		},
	}, nil, []runner.TaskFile{
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n", Mode: "0644"},
	})
	assert.Empty(t, environment)
	require.Len(t, files, 2)
	assert.Equal(t, "Implement the plan.\n", files[0].Content)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[1].Path)
}

func TestAttachWorkspaceAgentResourcesWritesSetupAndContext(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSetup, "setup", true, models.FactoryAgentResourceConfig{
		Commands: []string{"npm ci", "go mod download"},
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "superplanehq/suss",
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, []runner.TaskFile{
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n", Mode: "0644"},
	})

	paths := map[string]string{}
	for _, file := range files {
		paths[file.Path] = file.Content
	}
	require.Contains(t, paths, runner.WorkspaceSetupConfigPath)
	require.Contains(t, paths, runner.WorkspaceContextConfigPath)

	var setup struct {
		Commands []string `json:"commands"`
	}
	require.NoError(t, json.Unmarshal([]byte(paths[runner.WorkspaceSetupConfigPath]), &setup))
	assert.Equal(t, []string{"npm ci", "go mod download"}, setup.Commands)

	var contextFile struct {
		Repositories []struct {
			Repository string `json:"repository"`
			URL        string `json:"url"`
			Directory  string `json:"directory"`
		} `json:"repositories"`
	}
	require.NoError(t, json.Unmarshal([]byte(paths[runner.WorkspaceContextConfigPath]), &contextFile))
	require.Len(t, contextFile.Repositories, 1)
	assert.Equal(t, "superplanehq/suss", contextFile.Repositories[0].Repository)
	assert.Equal(t, "https://github.com/superplanehq/suss.git", contextFile.Repositories[0].URL)
	assert.Equal(t, "context/superplanehq/suss", contextFile.Repositories[0].Directory)
	assert.Contains(t, paths["prompts/implement.txt"], "Setup commands")
	assert.Contains(t, paths["prompts/implement.txt"], "superplanehq/suss (../context/superplanehq/suss)")
}

func TestAttachWorkspaceAgentResourcesSkipsSetupCommandsWhenConfigured(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSetup, "setup", true, models.FactoryAgentResourceConfig{
		Commands: []string{"npm ci"},
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "superplanehq/suss",
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration:  map[string]any{runner.SkipSetupCommandsField: true},
	}, nil, []runner.TaskFile{
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n", Mode: "0644"},
	})

	paths := map[string]string{}
	for _, file := range files {
		paths[file.Path] = file.Content
	}
	assert.Contains(t, paths, runner.WorkspaceCheckoutScriptPath)
	assert.Contains(t, paths, runner.WorkspaceContextConfigPath)
	assert.NotContains(t, paths, runner.WorkspaceSetupConfigPath)
	assert.NotContains(t, paths["prompts/implement.txt"], "npm ci")
	assert.Contains(t, paths["prompts/implement.txt"], "superplanehq/suss")
}

func TestAttachWorkspaceAgentResourcesAttachesSetupAndContextWithoutAgentFlag(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindSetup, "setup", true, models.FactoryAgentResourceConfig{
		Commands: []string{"npm ci"},
	})
	require.NoError(t, err)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "superplanehq/suss",
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)

	paths := map[string]string{}
	for _, file := range files {
		paths[file.Path] = file.Content
	}
	assert.Contains(t, paths, runner.WorkspaceSetupConfigPath)
	assert.Contains(t, paths, runner.WorkspaceContextConfigPath)
	assert.Contains(t, paths[runner.WorkspaceSetupConfigPath], "npm ci")
	assert.Contains(t, paths[runner.WorkspaceContextConfigPath], "superplanehq/suss")
}

func TestAttachWorkspaceAgentResourcesSkipsDisabledSetupAndContext(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	setup, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSetup, "setup", true, models.FactoryAgentResourceConfig{
		Commands: []string{"npm ci"},
	})
	require.NoError(t, err)
	repo, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "superplanehq/suss",
	})
	require.NoError(t, err)

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
		Configuration: map[string]any{
			"disabledAgentResourceIds": []string{setup.ID.String(), repo.ID.String()},
		},
	}, nil, []runner.TaskFile{
		{Path: "prompts/implement.txt", Content: "Implement the plan.\n", Mode: "0644"},
	})
	// The prompt stays untouched, but the checkout helper still ships: it
	// checks out the main repo even when every context repo is disabled.
	require.Len(t, files, 2)
	assert.Equal(t, "Implement the plan.\n", files[0].Content)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[1].Path)
}

func TestAttachWorkspaceAgentResourcesShipsCheckoutHelper(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")

	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	require.Len(t, files, 1)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, "0755", files[0].Mode)
	script := files[0].Content
	assert.Contains(t, script, "REPO_URL")
	assert.Contains(t, script, "workspace_context.json")
	assert.Contains(t, script, "git clone --depth 1")
	assert.Contains(t, script, runner.WorkspaceCheckoutCommitSetupEnv)
	assert.Contains(t, script, runner.FactoryRepoCommitSetup())
	assert.NotContains(t, script, "x-access-token")
	assert.NotContains(t, script, "insteadOf")
}

func TestAttachWorkspaceAgentResourcesShipsCheckoutHelperWithoutFlags(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")

	environment, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(),
		WorkflowID:     canvas.ID.String(),
	}, nil, nil)
	assert.Empty(t, environment)
	require.Len(t, files, 1)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
}

func TestAttachWorkspaceAgentResourcesKeepsSystemSkillsEnabled(t *testing.T) {
	r := setupWorkspaceAgentResources(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureWorkspaceSkills))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Line app")
	system, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "refine-task", true, models.FactoryAgentResourceConfig{
		Markdown: "# Refine task", SystemSkill: &models.FactorySystemSkill{TemplateID: "refine-task", Version: 1},
	})
	require.NoError(t, err)
	custom, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "custom", true, models.FactoryAgentResourceConfig{Markdown: "# Custom"})
	require.NoError(t, err)
	_, files := runner.AttachWorkspaceAgentResources(core.ExecutionContext{
		OrganizationID: r.Organization.ID.String(), WorkflowID: canvas.ID.String(),
		Configuration: map[string]any{"disabledAgentResourceIds": []string{system.ID.String(), custom.ID.String()}},
	}, nil, nil)
	require.Len(t, files, 3)
	assert.Equal(t, runner.WorkspaceCheckoutScriptPath, files[0].Path)
	assert.Equal(t, ".claude/skills/refine-task/SKILL.md", files[1].Path)
	assert.Equal(t, ".agents/skills/refine-task/SKILL.md", files[2].Path)
	assert.Equal(t, "# Refine task\n", files[1].Content)
	assert.Equal(t, files[1].Content, files[2].Content)
}
