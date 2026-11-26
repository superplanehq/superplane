package factories

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	canvasyaml "github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func TestFactoryConfigurationRepositoryRoundTrip(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	resource, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", false, models.FactoryAgentResourceConfig{Repository: "acme/docs", Ref: "release", Access: "read"})
	require.NoError(t, err)
	listener := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *canvasyaml.Canvas) {
		id := f.factory.OnboardingConfigValue().VCSIntegrationID
		for i := range doc.Spec.Nodes {
			if strings.HasPrefix(doc.Spec.Nodes[i].Component, "github.") {
				doc.Spec.Nodes[i].Integration = &canvasyaml.IntegrationRef{ID: id, Name: integrationName(t, f.r.Organization.ID, id)}
			}
		}
	})
	bundle := f.export(t)
	var manifest map[string]any
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	repositories, ok := manifest["repositories"].(map[string]any)
	require.True(t, ok, "export must include the workspace repositories")
	require.NotEmpty(t, repositories["binding"].(map[string]any)["integration_name"])
	additional := repositories["additional"].([]any)
	require.Len(t, additional, 1)
	require.Equal(t, false, additional[0].(map[string]any)["enabled"])
	require.Equal(t, "read", additional[0].(map[string]any)["access"])
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	additional[0].(map[string]any)["access"] = "write"
	additional[0].(map[string]any)["enabled"] = true
	additional[0].(map[string]any)["ref"] = "next"
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	saved, err := f.factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	require.True(t, saved.Enabled)
	require.Equal(t, "write", saved.Config.Data().Access)
	require.Equal(t, "next", saved.Config.Data().Ref)
	require.ElementsMatch(t, []any{"acme/app", "acme/docs"}, nodeWebhookRepositories(t, db, listener.ID, riskScoreTriggerNodeID))
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationResolvesNamedIntakeConnection(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	name := integrationName(t, f.r.Organization.ID, f.factory.OnboardingConfigValue().VCSIntegrationID)
	bundle.Files[0].Content = strings.Replace(bundle.Files[0].Content, "name: Backlog\n    automations: []", `name: Backlog
    automations:
      - id: issues
        name: Named connection intake
        uses: "@superplane/intake/github-issues"
        binding:
          integration_name: "`+name+`"`, 1)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	plan = f.plan(t, f.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationChangesRepositoriesAndIntakeTogether(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	previous := f.factory.OnboardingConfigValue()
	existing, err := f.factory.CreateWorkOrder(db, "Existing task", "", &f.r.User, nil, nil)
	require.NoError(t, err)
	// Legacy tasks inherit defaults until a repository change saves their pair.
	require.NoError(t, db.Model(existing).Updates(map[string]any{"repository": nil, "default_branch": nil, "vcs_provider": nil}).Error)
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{
		gitHubBranches: func(string) ([]string, error) { return []string{"main", "develop"}, nil },
	})
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Repositories.Primary = factoryConfigurationRepository{Repository: previous.BacklogRepository, Ref: "develop"}
		manifest.Repositories.BacklogRepository = previous.AppRepository
		manifest.Stages[0].Automations = append(manifest.Stages[0].Automations, factoryConfigurationAutomation{
			ID: "issues", Name: "Issues from desired repository", Uses: "@superplane/intake/github-issues",
		})
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	unchanged, err := f.factory.FindWorkOrder(db, existing.ID)
	require.NoError(t, err)
	require.Nil(t, unchanged.Repository, "plan must not snapshot tasks")
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	factory, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	config := factory.OnboardingConfigValue()
	require.Equal(t, previous.BacklogRepository, config.AppRepository)
	require.Equal(t, previous.AppRepository, config.BacklogRepository)
	require.Equal(t, "develop", config.DefaultBranch)
	saved, err := factory.FindWorkOrder(db, existing.ID)
	require.NoError(t, err)
	require.Equal(t, &previous.AppRepository, saved.Repository)
	require.Equal(t, &previous.DefaultBranch, saved.DefaultBranch)
	next, err := factory.CreateWorkOrder(db, "Next task", "", &f.r.User, nil, nil)
	require.NoError(t, err)
	require.Equal(t, &config.AppRepository, next.Repository)
	require.Equal(t, &config.DefaultBranch, next.DefaultBranch)

	canvas := configurationCanvasByName(t, f, "Issues from desired repository")
	live := resetTestLiveVersion(t, canvas)
	graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	trigger := findIntakeNode(live.Nodes, graph.TriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, config.BacklogRepository, trigger.Configuration["repository"])
	require.Equal(t, config.VCSIntegrationID, *trigger.IntegrationID)
	require.Equal(t, []string{config.BacklogRepository}, configurationWebhookRepositories(t, canvas, graph.TriggerNodeID))
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationCopiesRepositoriesToDestinationConnections(t *testing.T) {
	source := newConfigurationFixture(t)
	db := database.DB(t.Context())
	sourceID := source.factory.OnboardingConfigValue().VCSIntegrationID
	connectionName := integrationName(t, source.r.Organization.ID, sourceID)
	_, err := source.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", false, models.FactoryAgentResourceConfig{
		Repository: "acme/docs", Ref: "release", Access: models.FactoryContextRepoAccessRead,
	})
	require.NoError(t, err)
	sourceBundle := source.export(t)
	editConfigurationManifest(t, sourceBundle, func(manifest *factoryConfigurationManifest) {
		manifest.Stages[0].Automations = append(manifest.Stages[0].Automations, factoryConfigurationAutomation{
			ID: "issues", Name: "Portable issues", Uses: "@superplane/intake/github-issues",
			Binding: &factoryConfigurationBinding{IntegrationID: sourceID, ResourceID: "acme/backlog"},
		})
	})
	applyConfigurationBundle(t, source, sourceBundle)
	bundle := source.export(t)

	organization := support.CreateOrganization(t, source.r, source.r.User)
	destinationUser, err := models.CreateUserInTransaction(db, organization.ID, source.r.Account.ID, source.r.Account.Email, source.r.Account.Name)
	require.NoError(t, err)
	require.NoError(t, source.r.AuthService.AssignRole(destinationUser.ID.String(), models.RoleOrgAdmin, organization.ID.String(), models.DomainTypeOrganization))
	for _, feature := range []string{features.FeatureFactories, features.FeatureFactoryAsCode} {
		require.NoError(t, models.EnableExperimentalFeature(organization.ID, feature))
	}
	support.AssignOpenSystemTrial(t, organization.ID)
	destinationRegistry := *source.r
	destinationRegistry.Organization = organization
	destinationRegistry.User, destinationRegistry.UserModel = destinationUser.ID, destinationUser
	factory, err := models.CreateFactory(db, organization.ID, source.factory.Name, "", "")
	require.NoError(t, err)
	support.CompleteFactoryOnboarding(t, factory)
	destinationID := createReadyOnboardingIntegration(t, organization.ID, models.ProviderGitHub)
	require.NotEqual(t, sourceID, destinationID)
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", destinationID).Update("installation_name", connectionName).Error)
	repository, branch, agentID := "acme/backlog", "main", ""
	harness := models.FactoryOnboardingAgentHarnessSuperPlane
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &destinationID, AgentIntegrationID: &agentID, AgentHarness: &harness,
		AppRepository: &repository, BacklogRepository: &repository, DefaultBranch: &branch,
	}))
	canvas := resetTestTemplate(t, &destinationRegistry, factory, "line-implementation", models.ProviderGitHub, func(*canvasyaml.Canvas) {})
	line, err := factory.CreateLine(db, "Delivery", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: canvas.ID, Entrypoint: "onrun-implement"}})
	require.NoError(t, err)
	stages := line.StagesValue()
	for i := range stages {
		if stages[i].StepIndex != nil {
			stages[i].Name = "Build changes"
		}
	}
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	destination := configurationFixture{r: &destinationRegistry, factory: factory, canvas: canvas, line: line, ctx: authentication.SetUserIdInMetadata(source.ctx, destinationUser.ID.String())}
	destination.ctx = withRepositoryBranchResolver(destination.ctx, repositoryBranchResolver{
		gitHubBranches: func(string) ([]string, error) { return []string{"main"}, nil },
	})
	configureConfigurationCatalogIntegration(t, destination, destinationID, models.ProviderGitHub)
	applyConfigurationBundle(t, destination, bundle)

	saved, err := models.FindFactory(db, organization.ID, factory.ID)
	require.NoError(t, err)
	require.Equal(t, destinationID, saved.OnboardingConfigValue().VCSIntegrationID)
	require.Equal(t, source.factory.OnboardingConfigValue().AppRepository, saved.OnboardingConfigValue().AppRepository)
	repositories, err := saved.ListContextRepos(db)
	require.NoError(t, err)
	require.Len(t, repositories, 1)
	require.False(t, repositories[0].Enabled)
	require.Equal(t, "acme/docs", repositories[0].Config.Data().Repository)
	require.Equal(t, "release", repositories[0].Config.Data().Ref)
	require.Equal(t, models.FactoryContextRepoAccessRead, repositories[0].Config.Data().Access)
	intake := configurationCanvasByName(t, destination, "Portable issues")
	live := resetTestLiveVersion(t, intake)
	graph := resolveIntakeGraph(models.FactoryIntakeSourceGitHubIssues, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	trigger := findIntakeNode(live.Nodes, graph.TriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, destinationID, *trigger.IntegrationID)
	plan := destination.plan(t, destination.export(t))
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
}

func TestFactoryConfigurationRemovesOnlyDeclaredRepositories(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	config := f.factory.OnboardingConfigValue()
	repository, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/docs", Access: models.FactoryContextRepoAccessRead,
	})
	require.NoError(t, err)
	skill, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindSkill, "runbook", true, models.FactoryAgentResourceConfig{Markdown: "# Follow the runbook"})
	require.NoError(t, err)
	bundle := f.export(t)
	parallelism := 2
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Repositories = nil
		manifest.Stages[1].MaxParallelism = &parallelism
	})
	applyConfigurationBundle(t, f, bundle)
	saved, err := f.factory.FindAgentResource(db, repository.ID)
	require.NoError(t, err)
	require.Equal(t, repository, saved, "an absent section preserves additional repositories")
	factory, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, config, factory.OnboardingConfigValue())
	bundle = f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) { manifest.Repositories.Additional = nil })
	applyConfigurationBundle(t, f, bundle)
	repositories, err := f.factory.ListContextRepos(db)
	require.NoError(t, err)
	require.Empty(t, repositories)
	savedSkill, err := f.factory.FindAgentResource(db, skill.ID)
	require.NoError(t, err)
	require.Equal(t, skill, savedSkill, "removing repositories must preserve other agent resources")
}

func TestFactoryConfigurationRepositoryApplyRejectsStaleStateAndRollsBackFailure(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	original := f.factory.OnboardingConfigValue()
	resource, err := f.factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/docs", Ref: "main", Access: models.FactoryContextRepoAccessRead,
	})
	require.NoError(t, err)
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{
		gitHubBranches: func(string) ([]string, error) { return []string{"main", "develop"}, nil },
	})
	bundle := f.export(t)
	editConfigurationManifest(t, bundle, func(manifest *factoryConfigurationManifest) {
		manifest.Repositories.Primary.Ref = "develop"
		manifest.Repositories.Additional[0].Ref = "requested"
		manifest.Repositories.Additional = append(manifest.Repositories.Additional, factoryConfigurationContextRepository{Repository: "acme/new", Access: "read"})
		automation := &manifest.Stages[1].Automations[0]
		automation.With["agents"].(map[string]any)[implementationAgentNodeID].(map[string]any)["additional_instructions"] = "Publish these instructions."
	})
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	updated := resource.Config.Data()
	updated.Ref = "saved-in-ui"
	require.NoError(t, resource.Update(db, nil, nil, &updated))
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	saved, err := f.factory.FindAgentResource(db, resource.ID)
	require.NoError(t, err)
	require.Equal(t, "saved-in-ui", saved.Config.Data().Ref)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	versions := configurationRevisionCount(t, f.canvas)
	before := configurationResourceCounts(t)
	const callback = "configuration_repository_publication_failure"
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "workflow_nodes" {
			tx.AddError(errors.New("publication failed"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Update().Remove(callback) })
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.Error(t, err)
	factory, err := models.FindFactory(db, f.r.Organization.ID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, original, factory.OnboardingConfigValue())
	repositories, err := factory.ListContextRepos(db)
	require.NoError(t, err)
	require.Len(t, repositories, 1)
	require.Equal(t, resource.ID, repositories[0].ID)
	require.Equal(t, "saved-in-ui", repositories[0].Config.Data().Ref)
	require.Equal(t, before, configurationResourceCounts(t))
	require.Equal(t, versions, configurationRevisionCount(t, f.canvas))
}
