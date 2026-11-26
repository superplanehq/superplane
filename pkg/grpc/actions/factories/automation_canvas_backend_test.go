package factories

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func TestCanvasAutomationScheduleEditMatchesForm(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)

	ctx, deps := onboardingContext(r), onboardingDeps(r)
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
	// Keep both cron times away from the test's publication window.
	now := time.Now().UTC()
	zone := time.FixedZone("test schedule", 3600)
	schedules := []string{
		now.In(zone).Add(2*time.Hour).Format("4 15") + " * * *",
		now.In(zone).Add(4*time.Hour).Format("4 15") + " * * *",
	}
	deliveries := func(id uuid.UUID) map[string]*models.CanvasNodeRequest {
		t.Helper()
		live, err := models.FindLiveCanvasVersionInTransaction(db, id)
		require.NoError(t, err)
		pending := map[string]*models.CanvasNodeRequest{}
		for _, node := range live.Nodes {
			if node.ComponentName() != "schedule" {
				continue
			}
			request, err := models.FindPendingRequestForNode(db, id, node.ID)
			require.NoError(t, err)
			require.Equal(t, "1", node.Configuration["timezone"])
			pending[configString(node.Configuration, "cronExpression")] = request
		}
		require.Len(t, pending, len(schedules))
		return pending
	}
	var form map[string]*models.CanvasNodeRequest
	for _, author := range []string{"form", "yaml"} {
		settings := &pb.FactoryCustomAutomation_Settings{
			Agent: &pb.FactoryAppTemplateAgent{Component: models.SuperPlaneRunnerComponent},
			Steps: []*pb.FactoryCustomAutomation_Step{{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Check the changes."}},
		}
		for _, cron := range schedules {
			settings.Triggers = append(settings.Triggers, &pb.FactoryCustomAutomation_StartTrigger{
				StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
				Schedule:  &pb.FactoryCustomAutomation_Schedule{CronExpression: cron, UtcOffset: 1},
			})
		}
		name := "Scheduled check via " + author
		created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
			FactoryId: factory.ID.String(), ColumnKey: models.CanvasColumnKeyDone, Name: name, Settings: settings,
		})
		require.NoError(t, err)
		id := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
		// Simulate a pending delivery that was already due before the prompt edit.
		for _, request := range deliveries(id) {
			require.NoError(t, request.Reschedule(db, now.Add(-time.Minute)))
		}
		if author == "form" {
			settings.Steps[0].Prompt = "Review the changes."
			_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), AutomationId: id.String(), Name: name, Settings: settings,
			})
			require.NoError(t, err)
			form = deliveries(id)
			continue
		}
		source, err := service.Read(ctx, r.Organization.ID, id)
		require.NoError(t, err)
		doc, err := automationspec.Parse(source.YAML)
		require.NoError(t, err)
		require.Nil(t, doc.Agent)
		edited := strings.Replace(source.YAML, "Check the changes.", "Review the changes.", 1)
		require.NotEqual(t, source.YAML, edited)
		_, err = service.Update(ctx, r.Organization.ID, id, r.User, edited)
		require.NoError(t, err)

		yaml := deliveries(id)
		for _, cron := range schedules {
			require.Contains(t, form, cron)
			require.Contains(t, yaml, cron)
			require.WithinDuration(t, form[cron].RunAt, yaml[cron].RunAt, time.Second)
		}
	}
}

func TestCanvasAutomationCustomStageRoundTripAndAdmission(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	line, err := factory.CreateLine(db, "QA line", nil)
	require.NoError(t, err)
	stageID := uuid.New()
	stageKey := "custom-" + stageID.String()
	stages := line.StagesValue()
	stages = append(stages[:1], append([]models.FactoryLineStage{{ID: stageID, Key: stageKey, Name: "QA", Kind: models.FactoryLineStagePhase}}, stages[1:]...)...)
	require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: onboardingDeps(r)}}
	ctx := onboardingContext(r)
	source := "apiVersion: v2\nkind: Automation\nname: QA check\nruns_on_task: true\ntriggers:\n  - use: task_enters_stage\nsteps:\n  - name: Inspect\n    prompt: Check the changes.\n"
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: stageKey}
	invalid := placement
	invalid.Stage = "custom-" + uuid.NewString()
	_, err = service.Create(ctx, invalid, r.User, source)
	require.ErrorContains(t, err, "custom stage in this factory")
	created, err := service.Create(ctx, placement, r.User, source)
	require.NoError(t, err)
	read, err := service.Read(ctx, r.Organization.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, stageKey, read.Stage)
	require.Equal(t, source, read.YAML)
	updated, err := service.Update(ctx, r.Organization.ID, created.ID, r.User, strings.Replace(source, "Check the changes.", "Review the changes.", 1))
	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	foreignFactory, err := models.CreateFactory(db, r.Organization.ID, "Other factory", "", "")
	require.NoError(t, err)
	invalid.FactoryID = foreignFactory.ID
	invalid.Stage = stageKey
	_, err = service.Create(ctx, invalid, r.User, source)
	require.ErrorContains(t, err, "custom stage in this factory")
	_, err = service.Validate(ctx, placement, strings.Replace(source, "task_enters_stage", "pull_request_updated", 1))
	require.ErrorContains(t, err, "requires Verify")
	scheduledSource := "apiVersion: v2\nkind: Automation\nname: Scheduled QA\nruns_on_task: false\ntriggers:\n  - use: schedule\n    config:\n      cron: '0 9 * * *'\nsteps:\n  - name: Inspect\n    prompt: Check the changes.\n"
	for _, unsupported := range []string{
		scheduledSource,
		strings.Replace(source, "task_enters_stage", "task_leaves_stage", 1),
		strings.Replace(source, "  - use: task_enters_stage", "  - use: task_enters_stage\n  - use: manual", 1),
	} {
		_, err := service.Create(ctx, placement, r.User, unsupported)
		require.ErrorContains(t, err, "Keep one task_enters_stage or manual trigger")
	}
	legacyCanvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Existing QA", "existing-start")
	require.NoError(t, legacyCanvas.SetColumnKey(db, stageKey))
	version, err := models.FindLiveCanvasVersionInTransaction(db, legacyCanvas.ID)
	require.NoError(t, err)
	modernVersion, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	for _, node := range modernVersion.Nodes {
		if node.ID != "stage-run" {
			continue
		}
		node.ID = "existing-agent"
		node.Configuration["executionTimeoutSeconds"] = 7200
		version.Nodes = append(version.Nodes, node)
	}
	version.Edges = []models.Edge{{SourceID: "existing-start", TargetID: "existing-agent", Channel: "default"}}
	require.NoError(t, canvases.PublishGeneratedCanvasNodes(ctx, db, legacyCanvas, r.User, "Configure existing QA", version.Nodes, version.Edges, intakePublisherOptions(onboardingDeps(r), r.Organization.ID)))
	version, err = models.FindLiveCanvasVersionInTransaction(db, legacyCanvas.ID)
	require.NoError(t, err)
	legacyRead, err := service.Read(ctx, r.Organization.ID, legacyCanvas.ID)
	require.NoError(t, err)
	require.Contains(t, legacyRead.YAML, "task_enters_stage")
	unchanged, err := models.FindLiveCanvasVersionInTransaction(db, legacyCanvas.ID)
	require.NoError(t, err)
	require.Equal(t, version.ID, unchanged.ID, "opening does not publish")
	_, err = service.Update(ctx, r.Organization.ID, legacyCanvas.ID, r.User, strings.Replace(legacyRead.YAML, "Review the changes.", "Inspect the changes.", 1))
	require.NoError(t, err)
	preserved, err := models.FindLiveCanvasVersionInTransaction(db, legacyCanvas.ID)
	require.NoError(t, err)
	require.Equal(t, version.Edges, preserved.Edges)
	require.Len(t, preserved.Nodes, 2, "an unrelated edit adds no event triggers")
	for _, id := range []uuid.UUID{created.ID, legacyCanvas.ID} {
		original, err := service.Read(ctx, r.Organization.ID, id)
		require.NoError(t, err)
		manual := strings.Replace(original.YAML, "task_enters_stage", "manual", 1)
		_, err = service.Update(ctx, r.Organization.ID, id, r.User, manual)
		require.NoError(t, err)
		read, err := service.Read(ctx, r.Organization.ID, id)
		require.NoError(t, err)
		require.Equal(t, manual, read.YAML)
		_, err = service.Update(ctx, r.Organization.ID, id, r.User, original.YAML)
		require.NoError(t, err)
	}
	for _, name := range []string{"retired-secret", "kept-secret", "canvas-secret"} {
		_, err := models.CreateSecret(name, "local", r.User.String(), models.DomainTypeOrganization, r.Organization.ID, []byte("{}"))
		require.NoError(t, err)
	}
	for _, id := range []uuid.UUID{created.ID, legacyCanvas.ID} {
		t.Run(id.String(), func(t *testing.T) {
			bashSource := strings.Replace(source, "prompt: Check the changes.", "bash: make lint", 1)
			if id == legacyCanvas.ID {
				bashSource = strings.Replace(bashSource, "name: QA check", "name: Existing QA", 1)
			}
			withSettings := strings.Replace(bashSource, "runs_on_task: true", "runs_on_task: true\nskip_setup_commands: true\nskip_repository_checkout: true", 1)
			withSettings = strings.Replace(withSettings, "bash: make lint", "bash: make lint\n    secrets: [retired-secret, kept-secret]", 1)
			_, err := service.Update(ctx, r.Organization.ID, id, r.User, withSettings)
			require.NoError(t, err)
			version, err := models.FindLiveCanvasVersionInTransaction(db, id)
			require.NoError(t, err)
			for i := range version.Nodes {
				node := &version.Nodes[i]
				if node.ID == "stage-run" || node.ID == "existing-agent" {
					node.Configuration["environmentFrom"] = append(node.Configuration["environmentFrom"].([]any), map[string]any{"source": runner.EnvironmentFromSourceSecret, "secret": map[string]any{"secret": "canvas-secret"}})
				}
			}
			require.NoError(t, db.Save(version).Error)
			withoutSettings := strings.Replace(bashSource, "bash: make lint", "bash: make lint\n    secrets: [kept-secret]", 1)
			_, err = service.Update(ctx, r.Organization.ID, id, r.User, withoutSettings)
			require.NoError(t, err)
			version, err = models.FindLiveCanvasVersionInTransaction(db, id)
			require.NoError(t, err)
			for _, node := range version.Nodes {
				if !isFactoryAgentHarness(node.ComponentName()) {
					continue
				}
				require.NotContains(t, node.Configuration, "skipSetupCommands")
				require.NotContains(t, node.Configuration, "skipRepositoryCheckout")
				var secrets []string
				for _, source := range node.Configuration["environmentFrom"].([]any) {
					entry := source.(map[string]any)
					if entry["source"] == runner.EnvironmentFromSourceSecret {
						secrets = append(secrets, entry["secret"].(map[string]any)["secret"].(string))
					}
				}
				require.NotContains(t, secrets, "retired-secret")
				require.Contains(t, secrets, "kept-secret")
				if node.ID == "stage-run" || node.ID == "existing-agent" {
					require.Contains(t, secrets, "canvas-secret")
				}
			}
		})
	}
	require.NoError(t, legacyCanvas.SetColumnKey(db, ""))
	order, err := factory.CreateWorkOrder(db, "Check QA", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, order.TransitionOnDispatch(db, &r.User))
	repository, branch := "acme/task-repository", "task-changes"
	require.NoError(t, db.Model(order).Updates(map[string]any{"repository": repository, "default_branch": branch}).Error)
	var runs []*models.FactoryLineStepResult
	_, runs, err = line.DispatchStage(db, order)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "on-run", runs[0].Run.NodeID)
	require.Equal(t, order.ID.String(), runs[0].Run.Input.Data().(map[string]any)["work_order"].(map[string]any)["id"])
	version, err = models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	root := support.EmitCanvasEventForNodeWithData(t, created.ID, runs[0].Run.NodeID, "default", nil, runs[0].Run.Input.Data().(map[string]any))
	require.NoError(t, db.Model(root).Update("run_id", runs[0].Run.ID).Error)
	builder := contexts.NewNodeConfigurationBuilder(db, created.ID).WithRootEvent(&root.ID)
	for _, node := range version.Nodes {
		if node.ID != "stage-run" {
			continue
		}
		configuration, err := builder.Build(node.Configuration)
		require.NoError(t, err)
		environment := map[string]string{}
		for _, value := range configuration["environment"].([]any) {
			entry := value.(map[string]any)
			environment[entry["name"].(string)] = entry["value"].(string)
		}
		require.Equal(t, repository, environment["REPO"])
		require.Equal(t, "https://github.com/"+repository+".git", environment["REPO_URL"])
		require.Equal(t, branch, environment["BASE"])
	}
	current, err := service.Read(ctx, r.Organization.ID, created.ID)
	require.NoError(t, err)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, strings.Replace(current.YAML, "task_enters_stage", "manual", 1))
	require.NoError(t, err)
	started, err := RunFactoryCustomAutomation(ctx, onboardingDeps(r), r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: created.ID.String(), WorkOrderId: order.ID.String()})
	require.NoError(t, err)
	run, err := models.FindCanvasRunInTransaction(db, created.ID, uuid.MustParse(started.RunId))
	require.NoError(t, err)
	require.Equal(t, runs[0].Run.Input.Data(), run.Input.Data())
	require.Equal(t, "on-run", run.NodeID)
	var link models.FactoryWorkOrderRefinementRun
	require.NoError(t, db.Where("run_id = ?", run.ID).First(&link).Error)
	require.Equal(t, order.ID, link.WorkOrderID)
	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, created.ID)
	require.NoError(t, err)
	labels, err := models.ResolveCanvasRunTasks(db, canvas, []models.CanvasRun{*run})
	require.NoError(t, err)
	require.Equal(t, order.ID, labels[run.ID].ID)
	dispatch, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	require.NoError(t, dispatch.EnterStage(db, "verify"))
	_, err = RunFactoryCustomAutomation(ctx, onboardingDeps(r), r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: created.ID.String(), WorkOrderId: order.ID.String()})
	require.ErrorContains(t, err, "not in this stage")
}

func TestCanvasAutomationLegacyImportProtectsGraphBehavior(t *testing.T) {
	for _, tc := range []struct {
		name          string
		mode          pb.FactoryCustomAutomation_StartMode
		use           automationspec.TriggerUse
		runsOnTask    bool
		stage         string
		extraTriggers []*pb.FactoryCustomAutomation_StartTrigger
	}{
		{"manual", pb.FactoryCustomAutomation_START_MODE_MANUAL, automationspec.Manual, false, models.CanvasColumnKeyBacklog, nil},
		{"leave", pb.FactoryCustomAutomation_START_MODE_ON_LEAVE, automationspec.TaskLeavesStage, true, models.CanvasColumnKeyBacklog, nil},
		{"verify PR and leave", pb.FactoryCustomAutomation_START_MODE_AUTOMATIC, automationspec.PullRequestUpdated, true, models.CanvasColumnKeyVerify, []*pb.FactoryCustomAutomation_StartTrigger{{StartMode: pb.FactoryCustomAutomation_START_MODE_ON_LEAVE}}},
		{"verify leave and PR", pb.FactoryCustomAutomation_START_MODE_ON_LEAVE, automationspec.TaskLeavesStage, true, models.CanvasColumnKeyVerify, []*pb.FactoryCustomAutomation_StartTrigger{{StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			upsertHostedOnboardingProvider(t, db)
			r.Registry.Integrations[models.ProviderGitHub] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
				Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: prFeedbackTriggerComponent(models.ProviderGitHub)})},
			})
			ctx, deps := onboardingContext(r), onboardingDeps(r)
			triggers := []*pb.FactoryCustomAutomation_StartTrigger{{StartMode: tc.mode}}
			triggers = append(triggers, tc.extraTriggers...)
			runsOnTask := tc.runsOnTask
			skip := true
			const prompt = "  Inspect the task.\n"
			created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), ColumnKey: tc.stage, Name: "Inspect task",
				Settings: &pb.FactoryCustomAutomation_Settings{
					StartMode:              tc.mode,
					Triggers:               triggers,
					RunsOnTask:             &runsOnTask,
					SkipRepositoryCheckout: &skip,
					SkipSetupCommands:      &skip,
					Agent:                  &pb.FactoryAppTemplateAgent{Component: models.SuperPlaneRunnerComponent, Model: "hosted::anthropic::sonnet"},
					Steps: []*pb.FactoryCustomAutomation_Step{
						{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: prompt, WorkingDirectory: "/tmp"},
					},
				},
			})
			require.NoError(t, err)
			id := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
			backend := CanvasAutomationBackend{Deps: deps}
			legacy, err := backend.FindLegacy(ctx, db, r.Organization.ID, id)
			require.NoError(t, err)
			source, err := backend.ImportLegacy(ctx, db, legacy)
			require.NoError(t, err)
			doc, err := automationspec.Parse(source)
			require.NoError(t, err)
			require.Equal(t, &automationspec.Agent{Model: "sonnet"}, doc.Agent)
			require.NotNil(t, doc.RunsOnTask)
			require.Equal(t, tc.runsOnTask, *doc.RunsOnTask)
			require.Equal(t, tc.use, doc.Triggers[0].Use)
			require.Len(t, doc.Triggers, len(triggers))
			require.Equal(t, strings.TrimSpace(prompt), created.GetCustomAutomation().GetSettings().Steps[0].Prompt)
			wantPrompt := prompt
			if tc.stage == models.CanvasColumnKeyBacklog && !tc.runsOnTask {
				wantPrompt = strings.TrimSpace(prompt)
			}
			require.Equal(t, wantPrompt, doc.Steps[0].Prompt)
			require.Equal(t, "/tmp", doc.Steps[0].WorkingDirectory)
			require.Equal(t, &skip, doc.SkipSetupCommands)
			require.Equal(t, &skip, doc.SkipRepositoryCheckout)
			version, err := models.FindLiveCanvasVersionInTransaction(db, id)
			require.NoError(t, err)
			require.Equal(t, version.ID.String(), legacy.Artifact.Revision)
			customAutomationAgentForRunner(version.Nodes, customAutomationRunnerID).Configuration["extraCustomSetting"] = true
			require.NoError(t, db.Save(version).Error)
			_, err = backend.ImportLegacy(ctx, db, legacy)
			require.ErrorContains(t, err, "cannot be represented in YAML")

			delete(customAutomationAgentForRunner(version.Nodes, customAutomationRunnerID).Configuration, "extraCustomSetting")
			require.NoError(t, db.Save(version).Error)
			service := automations.Service{Backend: backend}
			adopted, err := service.Update(ctx, r.Organization.ID, id, r.User, source)
			require.NoError(t, err)
			require.Equal(t, id, adopted.ID)
			require.Equal(t, int64(1), adopted.Revision)
			if tc.mode == pb.FactoryCustomAutomation_START_MODE_MANUAL {
				started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
					FactoryId: factory.ID.String(), AutomationId: id.String(),
				})
				require.NoError(t, err)
				require.NotEmpty(t, started.GetRunId())
			}

			require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
			settings := created.GetCustomAutomation().GetSettings()
			skipWorkspaceSetup := false
			settings.SkipSetupCommands = &skipWorkspaceSetup
			settings.SkipRepositoryCheckout = &skipWorkspaceSetup
			const newPrompt = "Inspect the new task.\n"
			settings.Steps[0].Prompt = newPrompt
			updated, err := UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), AutomationId: id.String(), Name: "Inspect latest task", Settings: settings,
			})
			require.NoError(t, err)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
			reopened, err := service.Read(ctx, r.Organization.ID, id)
			require.NoError(t, err)
			doc, err = automationspec.Parse(reopened.YAML)
			require.NoError(t, err)
			require.Equal(t, "Inspect latest task", reopened.Name)
			require.Equal(t, strings.TrimSpace(newPrompt), updated.GetCustomAutomation().GetSettings().Steps[0].Prompt)
			wantPrompt = newPrompt
			if tc.stage == models.CanvasColumnKeyBacklog && !tc.runsOnTask {
				wantPrompt = strings.TrimSpace(newPrompt)
			}
			require.Equal(t, wantPrompt, doc.Steps[0].Prompt)
			require.Equal(t, int64(2), reopened.Revision)
			require.Equal(t, tc.use, doc.Triggers[0].Use)
			require.Len(t, doc.Triggers, len(triggers))
			if tc.name == "leave" {
				line, err := factory.CreateLine(db, "Move test", nil)
				require.NoError(t, err)
				_, err = UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
					FactoryId: factory.ID.String(), LineId: line.ID.String(), StageConfiguration: &pb.FactoryLine_StageConfiguration{
						Stages: factoryLineStageRequests(line.StagesValue()), AutomationMoves: []*pb.FactoryLine_AutomationMove{{Type: "canvas", AutomationId: id.String(), SourceKey: "backlog", DestinationKey: "done"}},
					},
				})
				require.NoError(t, err)
				moved, err := service.Read(ctx, r.Organization.ID, id)
				require.NoError(t, err)
				require.Equal(t, reopened.YAML, moved.YAML)
				require.Equal(t, reopened.Revision+1, moved.Revision)
				require.Equal(t, "done", moved.Stage)
			}
		})
	}
}

func TestAutomationFormFallbackRequiresVerifiedGraph(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	onTask := true
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL, RunsOnTask: &onTask,
		// The legacy form preserves custom icon keys outside the YAML icon picker.
		Icon: "legacy-icon", Agent: &pb.FactoryAppTemplateAgent{Component: "runnerCodex"},
		Steps: []*pb.FactoryCustomAutomation_Step{{Name: "Print", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "pwd"}},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), ColumnKey: models.CanvasColumnKeyImplement, Name: "Inspect task", Settings: settings,
	})
	require.NoError(t, err)
	id := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
	_, err = service.Read(ctx, r.Organization.ID, id)
	var fallback *AutomationFormFallback
	require.ErrorAs(t, err, &fallback)
	update := &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: id.String(), Name: "Renamed", Settings: settings,
		ExpectedLiveVersionId: fallback.LiveVersionID.String(),
	}
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), update)
	require.NoError(t, err)

	version, err := models.FindLiveCanvasVersionInTransaction(db, id)
	require.NoError(t, err)
	require.NotEqual(t, fallback.LiveVersionID, version.ID)
	version.Nodes = append(version.Nodes, models.Node{
		ID: "extra-action", Name: "Wait", Type: models.NodeTypeComponent,
		Ref: models.NodeRef{Component: &models.ComponentRef{Name: "timeGate"}},
	})
	version.Edges = append(version.Edges, models.Edge{SourceID: customAutomationRunnerID, TargetID: "extra-action", Channel: "default"})
	require.NoError(t, db.Save(version).Error)
	_, err = service.Read(ctx, r.Organization.ID, id)
	require.ErrorContains(t, err, "cannot be represented in YAML")
	require.NotErrorAs(t, err, &fallback)
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), update)
	require.ErrorContains(t, err, "Reopen it before you save")
}

func TestCanvasAutomationFollowsFactoryModelSource(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID, AgentIntegrationID: &agentID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-opus-5-5"})
	require.NoError(t, err)
	ctx := onboardingContext(r)
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: onboardingDeps(r)}}
	const source = `apiVersion: v2
kind: Automation
name: Inspect task
triggers:
  - use: manual
steps:
  - name: Inspect
    prompt: Inspect the task.
`
	created, err := service.Create(ctx, automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog}, r.User, source)
	require.NoError(t, err)
	live, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	require.Equal(t, "runnerClaudeCode", selectCustomAutomationAgent(live.Nodes).ComponentName())
	stubProviderModels(r.Registry, "openai", []string{"gpt-5"})
	changed, _, err := SwitchFactoryModelSourceInTransaction(ctx, r.Registry, r.Organization.ID.String(), factory.ID.String(), modelSourceOpenAI, "sk-test", "", "")
	require.NoError(t, err)
	require.Contains(t, changed, created.ID)
	live, err = models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	agent := selectCustomAutomationAgent(live.Nodes)
	require.NotNil(t, agent)
	require.Equal(t, "runnerCodex", agent.ComponentName())
	reopened, err := service.Read(ctx, r.Organization.ID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.YAML, reopened.YAML)
	require.Equal(t, created.Revision, reopened.Revision)
	explicit := strings.Replace(reopened.YAML, "triggers:", "agent: {harness: codex, model: gpt-5, credentials: {source: integration, name: openai}}\ntriggers:", 1)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, explicit)
	require.NoError(t, err)
	stubProviderModels(r.Registry, "claude", []string{"claude-opus-5-5"})
	_, _, err = SwitchFactoryModelSourceInTransaction(ctx, r.Registry, r.Organization.ID.String(), factory.ID.String(), modelSourceAnthropic, "sk-test", "", "")
	require.NoError(t, err)
	reopened, err = service.Read(ctx, r.Organization.ID, created.ID)
	require.NoError(t, err)
	doc, err := automationspec.Parse(reopened.YAML)
	require.NoError(t, err)
	require.Equal(t, "claude-code", string(doc.Agent.Harness))
	require.Equal(t, "claude-opus-5-5", doc.Agent.Model)
	connection, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(agentID))
	require.NoError(t, err)
	require.Equal(t, connection.InstallationName, doc.Agent.Credentials.Name)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, reopened.YAML)
	require.NoError(t, err)

	// Existing automations can use a connection different from the workspace default.
	otherID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	other, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(otherID))
	require.NoError(t, err)
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AgentIntegrationID: &otherID}))
	backend := CanvasAutomationBackend{Deps: onboardingDeps(r)}
	legacy, err := backend.FindLegacy(ctx, db, r.Organization.ID, created.ID)
	require.NoError(t, err)
	exported, err := backend.ImportLegacy(ctx, db, legacy)
	require.NoError(t, err)
	doc, err = automationspec.Parse(exported)
	require.NoError(t, err)
	require.Equal(t, automationspec.ClaudeCode, doc.Agent.Harness)
	require.Equal(t, connection.InstallationName, doc.Agent.Credentials.Name)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, strings.Replace(reopened.YAML, connection.InstallationName, other.InstallationName, 1))
	require.ErrorContains(t, err, "configured credentials")
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, created.YAML)
	require.NoError(t, err)
	live, err = models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	savedAgent, err := automationAgentDefinition(intakeAgentFromCanvasNodes(live.Nodes))
	require.NoError(t, err)
	require.Equal(t, connection.InstallationName, savedAgent.Credentials.Name)

	// A separate canvas publication must still protect its changes from a stale YAML save.
	edited, err := models.CreateCommitVersionWithSpecInTransaction(db, created.ID, r.User, "Edit in canvas", nil, nil)
	require.NoError(t, err)
	selectCustomAutomationAgent(edited.Nodes).Configuration["executionTimeoutSeconds"] = 900
	require.NoError(t, models.PromoteToLiveInTransaction(db, edited, edited.Nodes, edited.Edges))
	_, _, err = SwitchFactoryModelSourceInTransaction(ctx, r.Registry, r.Organization.ID.String(), factory.ID.String(), modelSourceOpenAI, "sk-test", "", "")
	require.NoError(t, err)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, created.YAML)
	require.ErrorContains(t, err, "automation changed")
}

func TestCanvasAutomationImportsSwitchedSubscription(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(connection).Updates(map[string]any{
		"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("test-bundle"),
		"available_models": datatypes.JSONSlice[models.CodexModel]{{Model: "gpt-5.4"}},
	}).Error)
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), ColumnKey: models.CanvasColumnKeyBacklog, Name: "Inspect task",
		Settings: &pb.FactoryCustomAutomation_Settings{
			StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
			Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerCodex", CredentialSource: "subscription"},
			Steps:     []*pb.FactoryCustomAutomation_Step{{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Inspect changes."}},
		},
	})
	require.NoError(t, err)
	id := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
	changed, _, err := SwitchFactoryModelSourceInTransaction(ctx, r.Registry, r.Organization.ID.String(), factory.ID.String(), modelSourceSubscription, "", "", "", r.User.String())
	require.NoError(t, err)
	require.Contains(t, changed, id)

	service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
	read, err := service.Read(ctx, r.Organization.ID, id)
	require.NoError(t, err)
	doc, err := automationspec.Parse(read.YAML)
	require.NoError(t, err)
	require.Nil(t, doc.Agent)
	read, err = service.Update(ctx, r.Organization.ID, id, r.User, read.YAML)
	require.NoError(t, err)
	doc.Agent = &automationspec.Agent{Model: "gpt-5.4"}
	selected, err := automationspec.Encode(doc.Definition)
	require.NoError(t, err)
	_, err = service.Validate(ctx, automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: read.Stage}, selected)
	require.NoError(t, err)
	_, err = service.Update(ctx, r.Organization.ID, id, r.User, selected)
	require.NoError(t, err)
	live, err := models.FindLiveCanvasVersionInTransaction(db, id)
	require.NoError(t, err)
	agent := intakeAgentFromCanvasNodes(live.Nodes)
	require.True(t, agent.subscription())
	require.Equal(t, "gpt-5.4", agent.Model)
	for _, changed := range []string{
		strings.Replace(selected, "gpt-5.4", "unavailable", 1),
		strings.Replace(selected, "model: gpt-5.4", "harness: superplane", 1),
		strings.Replace(selected, "model: gpt-5.4", "harness: codex\n  credentials: {source: hosted}", 1),
	} {
		_, err = service.Update(ctx, r.Organization.ID, id, r.User, changed)
		require.Error(t, err)
	}
	_, err = service.Update(ctx, r.Organization.ID, id, r.User, read.YAML)
	require.NoError(t, err)
	live, err = models.FindLiveCanvasVersionInTransaction(db, id)
	require.NoError(t, err)
	agent = intakeAgentFromCanvasNodes(live.Nodes)
	require.True(t, agent.subscription())
	require.Empty(t, agent.Model)
}

func TestCanvasAutomationResolvesAllowedDefaultModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID, AgentIntegrationID: &agentID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{"gpt-5-mini", "gpt-5-nano"})
	require.NoError(t, err)
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, datatypes.JSONSlice[string]{"gpt-5-nano"})
	require.NoError(t, err)
	connection, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(agentID))
	require.NoError(t, err)
	ctx := onboardingContext(r)
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: onboardingDeps(r)}}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog}
	const source = `apiVersion: v2
kind: Automation
name: Inspect task
triggers:
  - use: manual
steps:
  - name: Inspect
    prompt: Inspect the task.
`
	for _, tc := range []struct{ name, agent string }{
		{"named integration", "agent: {harness: codex, credentials: {source: integration, name: " + connection.InstallationName + "}}\n"},
		{"workspace agent", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := strings.Replace(source, "triggers:", tc.agent+"triggers:", 1)
			_, err := service.Validate(ctx, placement, yaml)
			require.NoError(t, err)
			created, err := service.Create(ctx, placement, r.User, yaml)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, service.Delete(ctx, r.Organization.ID, created.ID)) })
			live, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
			require.NoError(t, err)
			node := selectCustomAutomationAgent(live.Nodes)
			require.NotNil(t, node)
			require.Equal(t, "gpt-5-nano", node.Configuration["model"])
			resolved, err := models.ResolveBYOKRunnerModel(db, r.Organization.ID, &factory.ID, node.ComponentName(), node.Configuration)
			require.NoError(t, err)
			require.Equal(t, "gpt-5-nano", resolved)
		})
	}

	explicit := strings.Replace(source, "triggers:", "agent: {harness: codex, model: gpt-5-mini, credentials: {source: integration, name: "+connection.InstallationName+"}}\ntriggers:", 1)
	_, err = service.Validate(ctx, placement, explicit)
	require.ErrorContains(t, err, "agent.model")

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, nil)
	require.NoError(t, err)
	_, err = service.Validate(ctx, placement, source)
	require.ErrorContains(t, err, "agent.model")
	_, err = service.Create(ctx, placement, r.User, source)
	require.ErrorContains(t, err, "agent.model")
}
