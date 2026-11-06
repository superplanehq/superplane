package factories

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func setMultiRepoTasks(t *testing.T, r *support.ResourceRegistry, enabled bool) (BuiltInAutomationReset, error) {
	t.Helper()
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	return SetOrganizationMultiRepoTasks(ctx, r.Organization.ID, enabled, testIntakeDependencies(r))
}

func multiRepoTasksStored(t *testing.T, r *support.ResourceRegistry) bool {
	t.Helper()
	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	return slices.Contains(organization.EnabledExperimentalFeatures, features.FeatureFactoryMultiRepoTasks)
}

func hasPullRequestNode(nodes []models.Node) bool {
	return slices.ContainsFunc(nodes, func(node models.Node) bool { return isFactoryPullRequestComponent(node.ComponentName()) })
}

func TestSetOrganizationMultiRepoTasksSwitchesTheImplementTemplate(t *testing.T) {
	r, factory := resetTestFactory(t)
	github := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderGitHub, func(doc *yaml.Canvas) {
		setResetTestDraftPullRequests(t, doc, true)
	})
	bitbucket := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderBitbucket, func(doc *yaml.Canvas) {
		setResetTestDraftPullRequests(t, doc, true)
	})

	result, err := setMultiRepoTasks(t, r, true)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Reset, "both providers have a multi-repo variant")
	assert.True(t, multiRepoTasksStored(t, r))
	enabledVersions := map[*models.Canvas]uuid.UUID{}
	for _, canvas := range []*models.Canvas{github, bitbucket} {
		current := resetTestLiveVersion(t, canvas)
		assert.False(t, hasPullRequestNode(current.Nodes), "multi-repo agents open their own pull requests")
		assert.True(t, resetTestDraftPullRequests(t, current.Nodes))
		enabledVersions[canvas] = current.ID
	}

	result, err = setMultiRepoTasks(t, r, true)
	require.NoError(t, err)
	assert.Zero(t, result.Reset, "enabling twice must not publish again")
	for canvas, version := range enabledVersions {
		assert.Equal(t, version, resetTestLiveVersion(t, canvas).ID)
	}

	result, err = setMultiRepoTasks(t, r, false)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Reset)
	assert.False(t, multiRepoTasksStored(t, r))
	for _, canvas := range []*models.Canvas{github, bitbucket} {
		current := resetTestLiveVersion(t, canvas)
		assert.True(t, hasPullRequestNode(current.Nodes))
		assert.True(t, resetTestDraftPullRequests(t, current.Nodes))
	}
}

func TestSetOrganizationMultiRepoTasksRefreshesWorkspaceListeners(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "bitbucket", support.RandomName("bitbucket"), nil)
	require.NoError(t, err)
	factory := bitbucketFactory(t, r)
	integrationID := integration.ID.String()
	repository := "acme/api"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		AppRepository:    &repository,
	}))
	var setupRepositories [][]string
	r.Registry.Integrations["bitbucket"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyTrigger(impl.DummyTriggerOptions{
			Name: "bitbucket.onPullRequest",
			SetupFunc: func(ctx core.TriggerContext) error {
				repositories, err := ctx.Factory.ListRepositories()
				setupRepositories = append(setupRepositories, repositories)
				return err
			},
		})},
	})
	closure := support.CreateFactoryCanvas(t, r, factory.ID, "Close")
	publishRepositoryCanvas(t, r, testIntakeDependencies(r), closure, []models.Node{{
		ID: prClosureTriggerNodeID, Name: "On pull request closed", Type: models.NodeTypeTrigger,
		Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}},
		IntegrationID: &integrationID,
		Configuration: map[string]any{"repository": models.WorkspaceRepositoriesExpression, "actions": []any{"merged"}},
		Metadata:      models.FactoryAppTemplateMetadataFor("pr-closure", 1, models.ProviderBitbucket),
	}})
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository:  "acme/web",
		Access:      models.FactoryContextRepoAccessWrite,
		VCSProvider: models.ProviderBitbucket,
	})
	require.NoError(t, err)

	setupRepositories = nil
	_, err = setMultiRepoTasks(t, r, true)
	require.NoError(t, err)
	require.NotEmpty(t, setupRepositories, "enabling must refresh the workspace webhooks")
	assert.Equal(t, []string{"acme/api", "acme/web"}, setupRepositories[len(setupRepositories)-1])

	setupRepositories = nil
	_, err = setMultiRepoTasks(t, r, false)
	require.NoError(t, err)
	require.NotEmpty(t, setupRepositories, "disabling must refresh the workspace webhooks")
	assert.Equal(t, []string{"acme/api"}, setupRepositories[len(setupRepositories)-1])
}

func TestSetOrganizationMultiRepoTasksKeepsTheFlagWhenSettingsCannotApply(t *testing.T) {
	r, factory := resetTestFactory(t)
	canvas := resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderGitHub, func(*yaml.Canvas) {})
	version := resetTestLiveVersion(t, canvas)
	nodes := slices.Clone(version.Nodes)
	zero := 0
	for i := range nodes {
		if nodes[i].ID == implementationAgentNodeID {
			nodes[i].Concurrency = &models.ConcurrencySpec{Max: &zero}
		}
	}
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasVersion{}).
		Where("id = ?", version.ID).
		Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	_, err := setMultiRepoTasks(t, r, true)
	var settingsError *BuiltInAutomationSettingsError
	require.ErrorAs(t, err, &settingsError)
	assert.False(t, multiRepoTasksStored(t, r))
	current := resetTestLiveVersion(t, canvas)
	assert.Equal(t, version.ID, current.ID)
	assert.True(t, hasPullRequestNode(current.Nodes))
}
