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
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestReconcileFactoryRepositoryPublishesManagedRunnerCredentials(t *testing.T) {
	for _, testCase := range []struct {
		templateID           string
		provider             string
		runnerID             string
		repositoryExpression string
	}{
		{"line-implementation", models.ProviderBitbucket, implementationAgentNodeID, orderRepositoryExpression},
		{"risk-score", models.ProviderGitHub, "assess-risk", "new-workspace/api"},
	} {
		t.Run(testCase.templateID, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			previousID := createReadyOnboardingIntegration(t, r.Organization.ID, testCase.provider)
			selectedID := createReadyOnboardingIntegration(t, r.Organization.ID, testCase.provider)
			previousName := integrationName(t, r.Organization.ID, previousID)
			selectedName := integrationName(t, r.Organization.ID, selectedID)
			unrelatedID := createReadyOnboardingIntegration(t, r.Organization.ID, testCase.provider)
			unrelatedName := integrationName(t, r.Organization.ID, unrelatedID)
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			provider, repository, branch := testCase.provider, "new-workspace/api", "main"
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
				VCSProvider: &provider, VCSIntegrationID: &selectedID,
				AppRepository: &repository, BacklogRepository: &repository, DefaultBranch: &branch,
			}))
			canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Implement")
			nodes := []models.Node{
				{
					ID: "find-pr", Name: "Find Pull Request", Type: models.NodeTypeComponent,
					Ref:           models.NodeRef{Component: &models.ComponentRef{Name: testCase.provider + ".findPullRequest"}},
					IntegrationID: &previousID,
					Configuration: map[string]any{"repository": "old-workspace/api", "head": "task-branch"},
					Metadata:      models.FactoryAppTemplateMetadataFor(testCase.templateID, 1, testCase.provider),
				},
				{
					ID: testCase.runnerID, Name: "Implement", Type: models.NodeTypeComponent,
					Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerBash"}},
					Configuration: map[string]any{
						"machineType": "e1-large-amd64",
						"script":      "echo " + previousName,
						"environmentFrom": []any{
							map[string]any{"source": "integration", "integration": map[string]any{"name": previousName}},
							map[string]any{"source": "integration", "integration": map[string]any{"name": unrelatedName}},
						},
					},
				},
				{
					ID: "custom-runner", Name: "Clone another repository", Type: models.NodeTypeComponent,
					Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerBash"}},
					Configuration: map[string]any{
						"machineType": "e1-large-amd64",
						"script":      "git clone https://" + testCase.provider + ".org/other-workspace/tools.git",
						"environmentFrom": []any{
							map[string]any{"source": "integration", "integration": map[string]any{"name": previousName}},
						},
					},
				},
			}
			deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return canvases.PublishGeneratedCanvasNodes(t.Context(), tx, canvas, r.User, "Install implementation", nodes, nil,
					changesets.CanvasPublisherOptions{
						Registry: r.Registry, OrgID: r.Organization.ID, Encryptor: r.Encryptor,
						AuthService: r.AuthService, WebhookBaseURL: deps.WebhookBaseURL,
					})
			}))
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return reconcileFactoryRepository(t.Context(), tx, deps, factory, r.User,
					previousID, selectedID, "old-workspace/api", "old-workspace/api", branch, repository)
			}))
			reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
			require.NoError(t, err)
			version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, reloaded)
			require.NoError(t, err)
			require.Len(t, version.Nodes, 3)
			findPR := findIntakeNode(version.Nodes, "find-pr")
			managedRunner := findIntakeNode(version.Nodes, testCase.runnerID)
			custom := findIntakeNode(version.Nodes, "custom-runner")
			require.NotNil(t, findPR)
			require.NotNil(t, managedRunner)
			require.NotNil(t, custom)
			assert.Equal(t, selectedID, *findPR.IntegrationID)
			assert.Equal(t, testCase.repositoryExpression, findPR.Configuration["repository"])
			assert.Equal(t, []any{
				map[string]any{"source": "integration", "integration": map[string]any{"name": selectedName}},
				map[string]any{"source": "integration", "integration": map[string]any{"name": unrelatedName}},
			}, managedRunner.Configuration["environmentFrom"])
			assert.Equal(t, "echo "+previousName, managedRunner.Configuration["script"])
			assert.Equal(t, nodes[2].Configuration, custom.Configuration)
		})
	}
}

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

func TestReplaceVCSNodeIntegration(t *testing.T) {
	previousID := "old-integration"
	unrelatedID := "other-integration"
	nodes := []models.Node{
		{ID: "managed", Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}, Configuration: map[string]any{"repository": "acme/custom"}, IntegrationID: &previousID},
		{ID: "matching-repository", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, Configuration: map[string]any{"repository": "acme/old"}, IntegrationID: &previousID},
		{ID: "updated-repository", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, Configuration: map[string]any{"repository": "acme/new"}, IntegrationID: &previousID},
		{ID: "custom", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, Configuration: map[string]any{"repository": "acme/custom", "title": "acme/new"}, IntegrationID: &previousID},
		{ID: "other-integration", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "github.createIssue"}}, IntegrationID: &unrelatedID},
		{ID: "other-provider", Ref: models.NodeRef{Component: &models.ComponentRef{Name: "jira.createIssue"}}, IntegrationID: &previousID},
	}

	changed := replaceVCSNodeIntegration(
		nodes,
		map[string]bool{"managed": true},
		previousID,
		"new-integration",
		"acme/old",
		"acme/old",
		"acme/new",
	)

	assert.True(t, changed)
	assert.Equal(t, "new-integration", *nodes[0].IntegrationID)
	assert.Equal(t, "new-integration", *nodes[1].IntegrationID)
	assert.Equal(t, "new-integration", *nodes[2].IntegrationID)
	assert.Equal(t, previousID, *nodes[3].IntegrationID)
	assert.Equal(t, unrelatedID, *nodes[4].IntegrationID)
	assert.Equal(t, previousID, *nodes[5].IntegrationID)
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
