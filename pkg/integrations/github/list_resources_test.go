package github

import (
	"net/http"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func TestListRepositoryWebhooksForLimitRecovery(t *testing.T) {
	first := mocks.GitHubResponse(http.StatusOK, `[
		{"id":11,"events":["issues"],"config":{"url":"https://other.example/issues"}},
		{"id":12,"events":["pull_request"],"config":{"url":"https://user:password@old.example/hooks/path-token?token=secret#fragment-token"}}
	]`)
	first.Header.Set("Link", `<https://api.github.com/repos/acme/api/hooks?page=2>; rel="next"`)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{first, mocks.GitHubResponse(http.StatusOK, `[
		{"id":13,"events":["*"],"config":{"url":"https://current.example/webhook"}}
	]`)}}
	resources, err := (&GitHub{}).ListResources("repository_webhook", core.ListResourcesContext{
		HTTP: httpCtx, Integration: mocks.IntegrationContextForNewSetupFlow(),
		Parameters: map[string]string{"repository": "acme/api", "event": "pull_request"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2, "include every page and only hooks that use the limited event")
	require.Equal(t, "12", resources[0].ID)
	require.Contains(t, resources[0].Name, "old.example")
	require.NotContains(t, resources[0].Name, "/hooks/")
	require.NotContains(t, resources[0].Name, "path-token")
	require.NotContains(t, resources[0].Name, "fragment-token")
	require.NotContains(t, resources[0].Name, "secret")
	require.NotContains(t, resources[0].Name, "password")
	require.Equal(t, "https://github.com/acme/api/settings/hooks/12", resources[0].URL)
	require.Equal(t, "13", resources[1].ID)
}

func TestListResourcesHostedBindingReturnsOnlyGrantedRepositories(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	organization, err := models.CreateOrganization("resources-"+uuid.NewString(), "")
	require.NoError(t, err)
	require.NoError(t, models.UpsertVCSProviderInstallation(database.Conn(), &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: 501,
		AccountLogin:   "acme",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(database.Conn(), models.ProviderGitHub, 501, []models.VCSProviderRepository{
		{RepositoryID: 601, FullName: "acme/api"},
		{RepositoryID: 602, FullName: "acme/private"},
	}))
	integration, err := models.FindOrCreateVCSProviderBinding(database.Conn(), organization.ID, models.ProviderGitHub, 501, "acme")
	require.NoError(t, err)
	require.NoError(t, models.GrantVCSProviderBindingRepository(database.Conn(), integration.ID, models.ProviderGitHub, 601))

	resources, err := (&GitHub{}).ListResources("repository", core.ListResourcesContext{
		Integration: &contexts.IntegrationContext{
			IntegrationID: integration.ID.String(),
			Metadata:      common.Metadata{HostedApp: true},
			CurrentProperties: map[string]any{
				common.PropertyHost: "acme.ghe.com",
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "acme/api", resources[0].Name)
	assert.Equal(t, "601", resources[0].ID)
}

func Test__toIntegrationResources__usesFullName(t *testing.T) {
	fullName := "acme/web"
	shortName := "web"
	id := int64(42)

	resources := toIntegrationResources([]*github.Repository{
		{ID: &id, Name: &shortName, FullName: &fullName},
	})

	require.Len(t, resources, 1)
	assert.Equal(t, "repository", resources[0].Type)
	assert.Equal(t, "acme/web", resources[0].Name)
	assert.Equal(t, "42", resources[0].ID)
}

func Test__toIntegrationResources__fallsBackToName(t *testing.T) {
	shortName := "web"
	id := int64(7)

	resources := toIntegrationResources([]*github.Repository{
		{ID: &id, Name: &shortName},
	})

	require.Len(t, resources, 1)
	assert.Equal(t, "web", resources[0].Name)
}

func Test__toDefaultBranchResources__usesRepositoryDefaultBranch(t *testing.T) {
	for _, branch := range []string{"main", "master", "staging"} {
		defaultBranch := branch
		resources := toDefaultBranchResources(&github.Repository{DefaultBranch: &defaultBranch})

		require.Len(t, resources, 1)
		assert.Equal(t, "default_branch", resources[0].Type)
		assert.Equal(t, branch, resources[0].Name)
		assert.Equal(t, branch, resources[0].ID)
	}
}

func Test__toDefaultBranchResources__omitsAnEmptyDefaultBranch(t *testing.T) {
	assert.Empty(t, toDefaultBranchResources(&github.Repository{}))
	assert.Empty(t, toDefaultBranchResources(nil))
}

func Test__toLabelResources__usesLabelName(t *testing.T) {
	bug := "bug"
	needsTriage := "needs triage"

	resources := toLabelResources([]*github.Label{
		{Name: &bug},
		{Name: &needsTriage},
	})

	require.Len(t, resources, 2)
	assert.Equal(t, "label", resources[0].Type)
	assert.Equal(t, "bug", resources[0].Name)
	assert.Equal(t, "bug", resources[0].ID)
	assert.Equal(t, "needs triage", resources[1].Name)
}

func Test__toLabelResources__skipsLabelsWithoutName(t *testing.T) {
	empty := ""
	bug := "bug"

	resources := toLabelResources([]*github.Label{
		{},
		{Name: &empty},
		{Name: &bug},
	})

	require.Len(t, resources, 1)
	assert.Equal(t, "bug", resources[0].Name)
}
