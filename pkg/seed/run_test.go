package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/git/inmemory"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func TestRunRejectsIncompleteConfig(t *testing.T) {
	_, err := Run(context.Background(), &Dependencies{Config: Config{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hosted GitHub App is not configured")
}

func TestRunGitHubInstallationSelection(t *testing.T) {
	ctx := context.Background()

	t.Run("no installations", func(t *testing.T) {
		deps := newRunDeps(t, testSeedConfig(), nil)
		_, err := Run(ctx, deps)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoGitHubInstallations)
		assert.Contains(t, err.Error(), "Install the GitHub App")
	})

	t.Run("many installations require an id", func(t *testing.T) {
		cfg := testSeedConfig()
		deps := newRunDeps(t, cfg, []common.PendingInstallation{
			{ID: "11", AccountLogin: "acme"},
			{ID: "22", AccountLogin: "other"},
		})
		_, err := Run(ctx, deps)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SUPERPLANE_SEED_GITHUB_INSTALLATION_ID")
		assert.Contains(t, err.Error(), "11")
		assert.Contains(t, err.Error(), "22")
	})
}

func TestRunCreatesWorkspaceAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	cfg := testSeedConfig()
	cfg.InstallationID = "11"
	installs := []common.PendingInstallation{{ID: "11", AccountLogin: "acme"}}

	var listCalls, claudeCalls int
	deps := newRunDeps(t, cfg, installs)
	deps.ListInstallations = func(context.Context) ([]common.PendingInstallation, error) {
		listCalls++
		return installs, nil
	}
	deps.VerifyClaude = func(context.Context, string) error {
		claudeCalls++
		return nil
	}

	first, err := Run(ctx, deps)
	require.NoError(t, err)
	require.True(t, first.CreatedAccount)
	assert.Equal(t, cfg.Email, first.Email)
	assert.Equal(t, cfg.Password, first.Password)
	assert.Equal(t, "acme", first.GitHubOwner)
	assert.Equal(t, "acme/app", first.AppRepository)
	assert.Equal(t, "acme/app", first.BacklogRepository)
	assert.Equal(t, cfg.WorkspaceName, first.WorkspaceName)
	assert.NotEmpty(t, first.WorkspaceKey)
	assert.NotEmpty(t, first.OrganizationSlug)
	assert.Equal(t, 1, listCalls)
	assert.Equal(t, 1, claudeCalls)

	org, err := models.FindOrganizationBySlug(deps.DB, first.OrganizationSlug)
	require.NoError(t, err)
	listed, err := models.ListFactories(deps.DB, org.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.True(t, listed[0].IsOnboardingComplete())

	canvases, err := models.ListCanvases(org.ID.String())
	require.NoError(t, err)
	names := canvasNames(canvases)
	assert.Contains(t, names, "Plan")
	assert.Contains(t, names, "Implement")
	assert.Contains(t, names, "PR Closure")
	assert.Contains(t, names, "Create with an Agent")

	second, err := Run(ctx, deps)
	require.NoError(t, err)
	assert.False(t, second.CreatedAccount)
	assert.Empty(t, second.Password)
	assert.Equal(t, first.Email, second.Email)
	assert.Equal(t, first.OrganizationSlug, second.OrganizationSlug)
	assert.Equal(t, first.WorkspaceKey, second.WorkspaceKey)
	assert.Equal(t, 1, listCalls)
	assert.Equal(t, 1, claudeCalls)

	listedAgain, err := models.ListFactories(deps.DB, org.ID)
	require.NoError(t, err)
	assert.Len(t, listedAgain, 1)
	canvasesAgain, err := models.ListCanvases(org.ID.String())
	require.NoError(t, err)
	assert.Len(t, canvasesAgain, len(canvases))
}

func TestWriteSummaryHidesPasswordOnReuse(t *testing.T) {
	var buf bytes.Buffer
	WriteSummary(&buf, &Result{
		Email:             "dev@localhost",
		CreatedAccount:    false,
		OrganizationSlug:  "demo",
		WorkspaceName:     "Dev",
		WorkspaceKey:      "DEV",
		LoginURL:          "http://localhost:8000",
		AppRepository:     "acme/app",
		BacklogRepository: "acme/app",
	})
	assert.Contains(t, buf.String(), "unchanged")
	assert.NotContains(t, buf.String(), "superplane")
}

func testSeedConfig() Config {
	return Config{
		Email:            defaultEmail,
		Password:         defaultPassword,
		Name:             defaultName,
		OrganizationName: defaultOrganization,
		WorkspaceName:    defaultWorkspace,
		GitHubApp:        common.HostedApp{ID: 1, Slug: "superplane-dev", PrivateKey: "test-key", WebhookSecret: "whsec"},
		AnthropicAPIKey:  "sk-test",
		BaseURL:          "http://localhost:8000",
		WebhooksBaseURL:  "http://localhost:8000",
		DefaultBranch:    defaultBranchName,
	}
}

func newRunDeps(t *testing.T, cfg Config, installs []common.PendingInstallation) *Dependencies {
	t.Helper()
	require.NoError(t, database.TruncateTables())

	encryptor := crypto.NewNoOpEncryptor()
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)

	gitProvider := inmemory.NewProvider()
	provisioner := workers.NewRepositoryProvisionerWorker("", gitProvider)

	return &Dependencies{
		Config:      cfg,
		DB:          database.Conn(),
		Registry:    reg,
		Encryptor:   encryptor,
		AuthService: support.AuthService(t),
		GitProvider: gitProvider,
		ListInstallations: func(context.Context) ([]common.PendingInstallation, error) {
			return installs, nil
		},
		BindGitHub: func(_ context.Context, integration *models.Integration, installationID string) error {
			return bindTestGitHub(t, integration, installationID, "acme", []common.Repository{{Name: "app"}})
		},
		VerifyClaude: func(context.Context, string) error {
			return nil
		},
		ProvisionRepository: provisioner.ProvisionRepository,
	}
}

func bindTestGitHub(
	t *testing.T,
	integration *models.Integration,
	installationID, owner string,
	repos []common.Repository,
) error {
	t.Helper()
	metadata := common.Metadata{
		HostedApp:      true,
		InstallationID: installationID,
		Owner:          owner,
		Repositories:   repos,
		GitHubApp: common.GitHubAppMetadata{
			ID:   1,
			Slug: "superplane-dev",
		},
	}
	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	var asMap map[string]any
	require.NoError(t, json.Unmarshal(raw, &asMap))
	integration.Metadata = datatypes.NewJSONType(asMap)
	integration.State = models.IntegrationStateReady
	integration.StateDescription = ""
	require.NoError(t, database.Conn().Save(integration).Error)
	return nil
}

func canvasNames(canvases []models.Canvas) []string {
	names := make([]string, 0, len(canvases))
	for _, canvas := range canvases {
		names = append(names, canvas.Name)
	}
	return names
}

func TestRunMissingClaudeKey(t *testing.T) {
	cfg := testSeedConfig()
	cfg.AnthropicAPIKey = ""
	_, err := Run(context.Background(), &Dependencies{Config: cfg})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
}
