package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Plan/apply owns connection portability, persisted binding intent, and reset.
func TestFactoryConfigurationNamedBindingsSurviveExportAndReset(t *testing.T) {
	f := configurationCatalogFixture(t)
	templateID := intakeTemplateID(models.FactoryIntakeSourceLinearIssues)
	automation := configurationCatalogEntry(t, f, templateID)
	integrationID := automation.Binding.IntegrationID
	name := integrationName(t, f.r.Organization.ID, integrationID)
	automation.Binding.IntegrationName, automation.Binding.IntegrationID = name, ""
	bundle := configurationCatalogBundle(t, f, automation, templateID)
	applyConfigurationBundle(t, f, bundle)
	canvas := configurationCanvasByName(t, f, automation.Name)
	live := resetTestLiveVersion(t, canvas)
	graph := resolveIntakeGraph(models.FactoryIntakeSourceLinearIssues, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	trigger := findIntakeNode(live.Nodes, graph.TriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, integrationID, *trigger.IntegrationID)
	require.Equal(t, "team", trigger.Configuration["team"])
	owner, err := models.FindFactoryConfigurationAutomationByCanvas(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	require.Equal(t, integrationID, owner.Binding["integration_id"])
	require.NotContains(t, owner.Binding, "integration_name")

	exported := f.export(t)
	exportedAutomation := configurationManifestAutomation(t, exported, "backlog", automation.ID)
	require.Equal(t, &factoryConfigurationBinding{IntegrationName: name, TeamID: "team"}, exportedAutomation.Binding)
	plan := f.plan(t, exported)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)

	// ID and name references to one connection must compare equally.
	configurationUpdateCatalogAutomation(t, exported, automation.ID, func(candidate *factoryConfigurationAutomation) {
		candidate.Binding.IntegrationName, candidate.Binding.IntegrationID = "", integrationID
	})
	plan = f.plan(t, exported)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	revision := configurationRevisionCount(t, canvas)
	result, err := f.apply(exported, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, revision, configurationRevisionCount(t, canvas))

	require.NoError(t, resetTestOrganization(t, f.r))
	live = resetTestLiveVersion(t, canvas)
	graph = resolveIntakeGraph(models.FactoryIntakeSourceLinearIssues, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges})
	trigger = findIntakeNode(live.Nodes, graph.TriggerNodeID)
	require.Equal(t, integrationID, *trigger.IntegrationID)
	require.Equal(t, "team", trigger.Configuration["team"])
	require.Equal(t, []string{"project", "second"}, intakeSettingsFromGraph(models.FactoryIntakeSourceLinearIssues, graph, models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges}).LinearProjectIDs)
}

func TestFactoryConfigurationDiagnosesNamedConnectionsBeforeWrites(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, configurationFixture, *factoryConfigurationAutomation)
	}{
		{name: "missing name", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.IntegrationName = "Missing connection"
		}},
		{name: "case mismatch", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.IntegrationName = strings.ToUpper(automation.Binding.IntegrationName)
		}},
		{name: "another organization", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			other := support.CreateOrganization(t, f.r, f.r.User)
			id := createReadyOnboardingIntegration(t, other.ID, "jira")
			automation.Binding.IntegrationName = integrationName(t, other.ID, id)
		}},
		{name: "wrong provider", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			id := createReadyOnboardingIntegration(t, f.r.Organization.ID, "sentry")
			automation.Binding.IntegrationName = integrationName(t, f.r.Organization.ID, id)
		}},
		{name: "unavailable connection", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			integration, err := models.FindIntegrationByName(database.DB(t.Context()), f.r.Organization.ID, automation.Binding.IntegrationName)
			require.NoError(t, err)
			require.NoError(t, database.DB(t.Context()).Model(integration).Update("state", models.IntegrationStatePending).Error)
		}},
		{name: "unavailable capability", mutate: func(t *testing.T, f configurationFixture, automation *factoryConfigurationAutomation) {
			integration, err := models.FindIntegrationByName(database.DB(t.Context()), f.r.Organization.ID, automation.Binding.IntegrationName)
			require.NoError(t, err)
			require.NoError(t, database.DB(t.Context()).Model(integration).Update("capabilities", datatypes.NewJSONSlice([]models.CapabilityState{{Name: "jira.onIssue", State: core.IntegrationCapabilityStateDisabled}})).Error)
		}},
		{name: "name and ID", mutate: func(_ *testing.T, _ configurationFixture, automation *factoryConfigurationAutomation) {
			automation.Binding.IntegrationID = uuid.NewString()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := configurationCatalogFixture(t)
			templateID := intakeTemplateID(models.FactoryIntakeSourceJiraIssues)
			automation := configurationCatalogEntry(t, f, templateID)
			automation.Binding.IntegrationName = integrationName(t, f.r.Organization.ID, automation.Binding.IntegrationID)
			automation.Binding.IntegrationID = ""
			test.mutate(t, f, &automation)
			bundle := configurationCatalogBundle(t, f, automation, templateID)
			before := configurationResourceCounts(t)
			plan := f.plan(t, bundle)
			require.Len(t, plan.Diagnostics, 1)
			require.Equal(t, "stages[0].automations[0].binding.integration_name", plan.Diagnostics[0].Field)
			require.Empty(t, plan.BaseStateToken)
			_, err := f.apply(bundle, "invalid")
			require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			require.Equal(t, before, configurationResourceCounts(t))
		})
	}
}

func TestFactoryConfigurationUsesIDsForAmbiguousConnectionNames(t *testing.T) {
	f := configurationCatalogFixture(t)
	templateID := intakeTemplateID(models.FactoryIntakeSourceJiraIssues)
	automation := configurationCatalogEntry(t, f, templateID)
	integrationID := automation.Binding.IntegrationID
	bundle := configurationCatalogBundle(t, f, automation, templateID)
	applyConfigurationBundle(t, f, bundle)
	named := f.export(t)
	require.NotEmpty(t, configurationManifestAutomation(t, named, "backlog", automation.ID).Binding.IntegrationName)

	// The current schema enforces name uniqueness. Supply a legacy duplicate
	// at the database read boundary to check both wire-format fallbacks.
	db := database.DB(t.Context())
	callback := "configuration_ambiguous_connection_" + uuid.NewString()
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		integrations, ok := tx.Statement.Dest.(*[]models.Integration)
		if !ok {
			return
		}
		for _, integration := range *integrations {
			if integration.ID.String() == integrationID {
				integration.ID = uuid.New()
				*integrations = append(*integrations, integration)
				return
			}
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	exported := configurationManifestAutomation(t, f.export(t), "backlog", automation.ID).Binding
	require.Equal(t, integrationID, exported.IntegrationID)
	require.Empty(t, exported.IntegrationName)
	before := configurationResourceCounts(t)
	plan := f.plan(t, named)
	require.Len(t, plan.Diagnostics, 1)
	require.Equal(t, "stages[0].automations[0].binding.integration_name", plan.Diagnostics[0].Field)
	require.Contains(t, plan.Diagnostics[0].Message, "ambiguous")
	_, err := f.apply(named, "invalid")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationRestoresBitbucketDiscussionBinding(t *testing.T) {
	f := configurationCatalogFixture(t)
	db := database.DB(t.Context())
	provider := models.ProviderBitbucket
	integrationID := createReadyOnboardingIntegration(t, f.r.Organization.ID, provider)
	configureConfigurationCatalogIntegration(t, f, integrationID, provider)
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
	repository := f.factory.OnboardingConfigValue().BacklogRepository
	created, err := CreateFactoryPRFeedbackHandler(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: f.factory.ID.String(), Name: "Saved discussion repository",
		Settings: &pb.FactoryPRFeedbackHandler_Settings{Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{Repository: repository}},
	})
	require.NoError(t, err)
	require.Equal(t, repository, created.GetHandler().GetSettings().GetSubject().GetRepository())
	exported := f.export(t)
	_, err = DeleteFactoryPRFeedbackHandler(f.ctx, f.factory.OrganizationID.String(), &pb.DeleteFactoryPRFeedbackHandlerRequest{
		FactoryId: f.factory.ID.String(), HandlerId: created.GetHandler().GetId(),
	})
	require.NoError(t, err)
	applyConfigurationBundle(t, f, exported)
	canvas := configurationCanvasByName(t, f, "Saved discussion repository")
	for _, reset := range []bool{false, true} {
		if reset {
			require.NoError(t, resetTestOrganization(t, f.r))
		}
		live := resetTestLiveVersion(t, canvas)
		spec := models.LiveCanvasSpec{Nodes: live.Nodes, Edges: live.Edges}
		require.Equal(t, repository, prFeedbackSettingsFromGraph(resolvePRFeedbackGraph(spec), spec).Repository)
	}
}
