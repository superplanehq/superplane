package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/gorm"
)

func TestFactoryConfigurationCustomDispatchRoundTrip(t *testing.T) {
	for _, placement := range []string{"phase-0", "custom-" + uuid.NewString()} {
		t.Run(placement, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderGitHub)
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			upsertHostedOnboardingProvider(t, db)
			ctx, deps := onboardingContext(r), onboardingDeps(r)
			feature := features.FeatureFactoryCustomAutomations
			if strings.HasPrefix(placement, "custom-") {
				feature = features.FeatureFactoryCustomStages
			}
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, feature))
			const source = `apiVersion: v2
kind: Automation
name: Inspect task
triggers:
  - use: manual
steps:
  - name: Inspect
    prompt: Inspect the task.
`
			document, err := automationspec.Parse(source)
			require.NoError(t, err)
			nodes, edges, err := (CanvasAutomationBackend{Deps: deps}).compile(db, automations.Placement{
				OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: placement,
			}, document, nil, r.User)
			require.NoError(t, err)
			// A bound runner can predate the compiler's default trigger identity.
			for i := range nodes {
				if nodes[i].ComponentName() == "onRun" {
					nodes[i].ID = "existing-dispatch-trigger"
				}
			}
			edges[0].SourceID = "existing-dispatch-trigger"
			id := uuid.New()
			_, err = canvases.CreateCanvasWithIDInTransaction(ctx, db, id, deps.Registry, deps.Encryptor, deps.AuthService,
				deps.WebhookBaseURL, factory.OrganizationID, document.Name, "", &factory.ID, nodes, edges)
			require.NoError(t, err)
			canvas, err := models.FindCanvasInTransaction(db, factory.OrganizationID, id)
			require.NoError(t, err)
			if strings.HasPrefix(placement, "custom-") {
				require.NoError(t, canvas.SetColumnKey(db, placement))
			}
			live, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
			require.NoError(t, err)
			var entrypoint string
			for _, node := range live.Nodes {
				if node.ComponentName() == "onRun" {
					entrypoint = node.ID
				}
			}
			require.NotEmpty(t, entrypoint)
			exported, err := exportFactoryConfigurationCustom(db, factory, canvas, live, placement)
			require.NoError(t, err)
			edited := strings.Replace(exported, "Inspect the task.", "Review the task.", 1)
			require.NotEqual(t, exported, edited)
			require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, feature))
			_, err = validateFactoryConfigurationCustom(ctx, db, deps, factory, canvas, live, placement, exported)
			require.NoError(t, err, "an unchanged export remains valid when its creation feature is disabled")
			_, err = validateFactoryConfigurationCustom(ctx, db, deps, factory, canvas, live, placement, edited)
			require.ErrorContains(t, err, feature)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, feature))
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, err := applyFactoryConfigurationCustom(ctx, tx, deps, factory, canvas, live, placement, edited, r.User)
				return err
			}))
			live, err = models.FindLiveCanvasVersionInTransaction(db, id)
			require.NoError(t, err)
			require.NotNil(t, findIntakeNode(live.Nodes, entrypoint))
			require.Equal(t, "onRun", findIntakeNode(live.Nodes, entrypoint).ComponentName())
			reexported, err := exportFactoryConfigurationCustom(db, factory, canvas, live, placement)
			require.NoError(t, err)
			require.Contains(t, reexported, "Review the task.")
			stored, err := models.FindAutomationSourceByArtifact(db, factory.OrganizationID, id.String())
			require.NoError(t, err)
			require.Equal(t, live.ID.String(), stored.ArtifactRevision)
			_, err = validateFactoryConfigurationCustom(ctx, db, deps, factory, canvas, live, placement,
				strings.Replace(edited, "use: manual", "use: task_enters_stage", 1))
			require.ErrorContains(t, err, "manual")
			_, err = validateFactoryConfigurationCustom(ctx, db, deps, factory, canvas, live, placement,
				strings.Replace(edited, "prompt: Review the task.", "bash: echo ready\n    secrets: [missing-secret]", 1))
			require.ErrorContains(t, err, "steps[0].secrets[0]")
			selectCustomAutomationAgent(live.Nodes).Configuration["unsupportedSetting"] = true
			_, err = exportFactoryConfigurationCustom(db, factory, canvas, live, placement)
			require.ErrorContains(t, err, canvas.Name)
		})
	}
}

func TestFactoryConfigurationCustomBundleCoversEveryPlacement(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	deps := testIntakeDependencies(f.r)
	for _, feature := range []string{features.FeatureFactoryCustomAutomations, features.FeatureFactoryCustomStages} {
		require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, feature))
	}
	vcsID := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.ProviderGitHub)
	repository, branch := "acme/app", "main"
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID, AppRepository: &repository, DefaultBranch: &branch,
	}))
	support.AssignOpenSystemTrial(t, f.r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	f.r.Registry.Integrations[models.ProviderGitHub] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: prFeedbackTriggerComponent(models.ProviderGitHub)})},
	})
	customStage := models.FactoryLineStage{ID: uuid.New(), Key: "custom-" + uuid.NewString(), Name: "Security", Kind: models.FactoryLineStageCustom}
	placements := []string{"backlog", "implement", "verify", "done", "phase-1", customStage.Key}
	canvasesByPlacement := make(map[string]*models.Canvas, len(placements))
	for _, placement := range placements {
		source := "apiVersion: v2\nkind: Automation\nname: Inspect " + placement + "\nruns_on_task: true\ntriggers:\n  - use: manual\nsteps:\n  - name: Inspect\n    prompt: Inspect the task.\n"
		document, err := automationspec.Parse(source)
		require.NoError(t, err)
		nodes, edges, err := (CanvasAutomationBackend{Deps: deps}).compile(db, automations.Placement{
			OrganizationID: f.factory.OrganizationID, FactoryID: f.factory.ID, Stage: placement,
		}, document, nil, f.r.User)
		require.NoError(t, err)
		id := uuid.New()
		_, err = canvases.CreateCanvasWithIDInTransaction(f.ctx, db, id, deps.Registry, deps.Encryptor, deps.AuthService,
			deps.WebhookBaseURL, f.factory.OrganizationID, document.Name, "", &f.factory.ID, nodes, edges)
		require.NoError(t, err)
		canvas, err := models.FindCanvasInTransaction(db, f.factory.OrganizationID, id)
		require.NoError(t, err)
		if !strings.HasPrefix(placement, "phase-") {
			require.NoError(t, canvas.SetColumnKey(db, placement))
		}
		canvasesByPlacement[placement] = canvas
	}
	steps := append([]models.FactoryLineStep{}, f.line.Steps...)
	steps = append(steps, models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: canvasesByPlacement["phase-1"].ID, Entrypoint: "on-run"})
	require.NoError(t, f.line.Update(db, nil, steps, nil))
	stages := f.line.StagesValue()
	stages = append(stages[:3], append([]models.FactoryLineStage{customStage}, stages[3:]...)...)
	require.NoError(t, f.line.UpdateConfiguration(db, nil, nil, nil, stages))
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
	})
	require.NoError(t, err)
	bundle := exported.Bundle
	require.Len(t, bundle.Files, len(placements)+1)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	unchanged, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, unchanged.Changed)
	require.Equal(t, before, configurationResourceCounts(t))
	desiredSources := map[string]string{}
	for _, file := range bundle.Files[1:] {
		require.Contains(t, file.Content, "Inspect the task.")
		document, err := automationspec.Parse(file.Content)
		require.NoError(t, err)
		document.Agent = nil
		document.Steps[0].Prompt = "Review the task."
		source, err := automationspec.Encode(document.Definition)
		require.NoError(t, err)
		file.Content = "# Inherit the workspace agent.\n" + source
		desiredSources[file.Path] = file.Content
	}
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, len(placements))
	changed, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, changed.Changed)
	after := configurationResourceCounts(t)
	require.Equal(t, before["workflow_versions"]+int64(len(placements)), after["workflow_versions"])
	require.Equal(t, before["automations_as_code"]+int64(len(placements)), after["automations_as_code"])
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	unchanged, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, unchanged.Changed)
	require.Equal(t, after, configurationResourceCounts(t))
	exported, err = ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
		FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
	})
	require.NoError(t, err)
	for _, file := range exported.Bundle.Files[1:] {
		require.Equal(t, desiredSources[file.Path], file.Content)
		document, err := automationspec.Parse(file.Content)
		require.NoError(t, err)
		require.Nil(t, document.Agent)
	}
}
