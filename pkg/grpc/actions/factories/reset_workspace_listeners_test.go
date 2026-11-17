package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func TestSelectedImplementResetRefreshesWorkspaceListeners(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "bitbucket", support.RandomName("bitbucket"), nil)
	require.NoError(t, err)
	factory := bitbucketFactory(t, r)
	resetTestTemplate(t, r, factory, lineImplementationTemplateID, models.ProviderBitbucket, func(*yaml.Canvas) {})
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
	}, {
		ID: "complete-work-order", Name: "Complete task", Type: models.NodeTypeComponent,
		Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "updateWorkOrderStatus"}},
		Configuration: map[string]any{"state": "closed", "result": "completed", allPullRequestsDoneField: false},
	}})
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web", Access: models.FactoryContextRepoAccessWrite, VCSProvider: models.ProviderBitbucket,
	})
	require.NoError(t, err)
	setupRepositories = nil
	result, err := resetBuiltInAutomationForOrganizations(t.Context(), []models.Organization{*r.Organization}, lineImplementationTemplateID, testIntakeDependencies(r))
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reset)
	require.NotEmpty(t, setupRepositories)
	assert.Equal(t, []string{"acme/api", "acme/web"}, setupRepositories[len(setupRepositories)-1])
	complete := findIntakeNode(resetTestLiveVersion(t, closure).Nodes, "complete-work-order")
	require.NotNil(t, complete)
	assert.Equal(t, false, complete.Configuration[allPullRequestsDoneField])
}
