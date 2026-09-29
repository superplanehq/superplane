package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestUpdateFactoryRepositorySynchronizesHostedBindingAccess(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, "test-private-key")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")

	r := support.Setup(t)
	db := database.DB(t.Context())
	const githubUserID = int64(42)
	const installationID = int64(101)
	repositories := []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/api", DefaultBranch: "main"},
		{RepositoryID: 202, FullName: "acme/web", DefaultBranch: "trunk"},
	}

	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		r.Account.ID,
		models.ProviderGitHub,
		"42",
		"octocat",
		"The Octocat",
		"https://github.com/octocat.png",
	)))
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider:       models.ProviderGitHub,
		InstallationID: installationID,
		AccountLogin:   "acme",
	}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, repositories))
	for _, repository := range repositories {
		require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(
			db,
			models.ProviderGitHub,
			repository.RepositoryID,
			[]models.VCSProviderRepositoryCollaborator{{ProviderUserID: githubUserID, ProviderLogin: "octocat"}},
		))
	}

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	selected, err := SelectFactoryVCSProviderRepository(ctx, r.Organization.ID.String(), &pb.SelectFactoryVCSProviderRepositoryRequest{
		Id:           factory.ID.String(),
		Provider:     models.ProviderGitHub,
		RepositoryId: 201,
	})
	require.NoError(t, err)

	updated, err := UpdateFactoryRepository(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.UpdateFactoryRepositoryRequest{
		Id:            factory.ID.String(),
		Repository:    "ACME/WEB",
		DefaultBranch: "untrusted-branch",
	})
	require.NoError(t, err)
	assert.Equal(t, "acme/web", updated.Factory.Onboarding.AppRepository)
	assert.Equal(t, int64(202), updated.Factory.Onboarding.AppRepositoryId)
	assert.Equal(t, "trunk", updated.Factory.Onboarding.DefaultBranch)

	integrationID := selected.Factory.Onboarding.VcsIntegrationId
	granted, err := models.ListVCSProviderBindingRepositories(db, uuid.MustParse(integrationID))
	require.NoError(t, err)
	require.Len(t, granted, 1)
	assert.Equal(t, int64(202), granted[0].RepositoryID)
}

func TestReplaceConfigurationValues(t *testing.T) {
	configuration := map[string]any{
		"repository": "acme/old",
		"environment": []any{
			map[string]any{"name": "REPO", "value": "acme/old"},
			map[string]any{"name": "BASE", "value": "main"},
		},
		"command": "git clone --branch ${BASE:-main} https://github.com/acme/main-service.git",
	}

	replaced, changed := replaceConfigurationValues(configuration, []configurationReplacement{
		{from: "acme/old", to: "{{ task().repository }}"},
		{from: "main", to: "{{ task().default_branch }}"},
	})

	assert.True(t, changed)
	assert.Equal(t, "acme/old", configuration["repository"])
	assert.Equal(t, "{{ task().repository }}", replaced.(map[string]any)["repository"])
	assert.Equal(t, "{{ task().default_branch }}", replaced.(map[string]any)["environment"].([]any)[1].(map[string]any)["value"])
	assert.Equal(t, configuration["command"], replaced.(map[string]any)["command"])
}

func TestReplaceTriggerRepository(t *testing.T) {
	nodes := []models.Node{
		{Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/old"}},
		{Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/custom"}},
	}

	changed := replaceTriggerRepository(nodes, "github.onIssue", "acme/old", "acme/new")

	assert.True(t, changed)
	assert.Equal(t, "acme/new", nodes[0].Configuration["repository"])
	assert.Equal(t, "acme/custom", nodes[1].Configuration["repository"])
}

func TestReplaceGitHubNodeIntegration(t *testing.T) {
	previousID := "old-integration"
	unrelatedID := "other-integration"
	nodes := []models.Node{
		{ID: "managed", Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/custom"}, IntegrationID: &previousID},
		{ID: "matching-repository", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, Configuration: map[string]any{"repository": "acme/old"}, IntegrationID: &previousID},
		{ID: "custom", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, Configuration: map[string]any{"repository": "acme/custom"}, IntegrationID: &previousID},
		{ID: "other-integration", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, IntegrationID: &unrelatedID},
		{ID: "other-provider", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "jira.createIssue"}}, IntegrationID: &previousID},
	}

	changed := replaceGitHubNodeIntegration(
		nodes,
		map[string]bool{"managed": true},
		previousID,
		"new-integration",
		"acme/old",
		"acme/old",
	)

	assert.True(t, changed)
	assert.Equal(t, "new-integration", *nodes[0].IntegrationID)
	assert.Equal(t, "new-integration", *nodes[1].IntegrationID)
	assert.Equal(t, previousID, *nodes[2].IntegrationID)
	assert.Equal(t, unrelatedID, *nodes[3].IntegrationID)
	assert.Equal(t, previousID, *nodes[4].IntegrationID)
}

func TestFactoryDefaultBranchFromNodes(t *testing.T) {
	nodes := []models.Node{
		{
			Configuration: map[string]any{
				"environment": []any{
					map[string]any{"name": "BASE", "value": "develop"},
				},
			},
		},
	}

	assert.Equal(t, "develop", factoryDefaultBranchFromNodes(nodes))
}

func TestFactoryDefaultBranchFromNodesIgnoresExpressions(t *testing.T) {
	nodes := []models.Node{
		{Configuration: map[string]any{"base": orderDefaultBranchExpression}},
	}

	assert.Empty(t, factoryDefaultBranchFromNodes(nodes))
}
