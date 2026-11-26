package factories

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestUpdateFactoryLineStages(t *testing.T) {
	r := support.Setup(t)
	deps := testIntakeDependencies(r)
	db := database.DB(t.Context())

	newLine := func(t *testing.T) (*models.Factory, *models.FactoryLine) {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		line, err := factory.CreateLine(db, "ship", nil)
		require.NoError(t, err)
		return factory, line
	}

	t.Run("adds, reorders, renames, colors, and removes custom stages", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{
			{Key: "new-security", Name: " Security review ", Kind: pb.FactoryLine_STAGE_KIND_PHASE},
			{Key: "new-release", Name: "Release", Kind: pb.FactoryLine_STAGE_KIND_PHASE},
		}, stages[1:]...)...)

		created, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnColors: map[string]string{
				"new-security": "purple",
			},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{
				Stages: stages,
			},
		})
		require.NoError(t, err)
		require.Len(t, created.Line.Stages, 5)
		for _, stage := range created.Line.Stages[1:4] {
			assert.Equal(t, pb.FactoryLine_STAGE_KIND_PHASE, stage.Kind)
		}
		security := created.Line.Stages[1]
		release := created.Line.Stages[2]
		assert.Equal(t, "Security review", security.Name)
		assert.NotEmpty(t, security.Id)
		assert.Contains(t, security.Key, "custom-")
		assert.True(t, security.GetAutoAdvance())
		assert.True(t, release.GetAutoAdvance())
		assert.False(t, created.Line.Stages[3].GetAutoAdvance())
		assert.Equal(t, "purple", created.Line.ColumnColors[security.Key])

		reordered := []*pb.FactoryLine_Stage{
			created.Line.Stages[0],
			release,
			{Id: security.Id, Key: security.Key, Name: "Security approval", Kind: security.Kind},
			created.Line.Stages[3],
			created.Line.Stages[4],
		}
		updated, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnColors: map[string]string{
				security.Key: "purple",
			},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: reordered},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Security approval", "Verify", "Done"}, stageNames(updated.Line.Stages))
		assert.Equal(t, "purple", updated.Line.ColumnColors[security.Key])

		withoutSecurity := append([]*pb.FactoryLine_Stage{}, updated.Line.Stages[:2]...)
		withoutSecurity = append(withoutSecurity, updated.Line.Stages[3:]...)
		removed, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: withoutSecurity},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Verify", "Done"}, stageNames(removed.Line.Stages))
		assert.NotContains(t, removed.Line.ColumnColors, security.Key)

		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Release", "Verify", "Done"}, modelStageNames(reloaded.StagesValue()))
	})

	t.Run("persists auto advance and preserves it when omitted", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		autoAdvance := false
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{{
			Key: "new-qa", Name: "QA", Kind: pb.FactoryLine_STAGE_KIND_PHASE, AutoAdvance: &autoAdvance,
		}}, stages[1:]...)...)

		created, err := UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)
		qa := created.Line.Stages[1]
		assert.False(t, qa.GetAutoAdvance())

		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		withoutSetting := factoryLineStageRequests(reloaded.StagesValue())
		withoutSetting[1].AutoAdvance = nil
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: withoutSetting},
		})
		require.NoError(t, err)
		reloaded, err = factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.False(t, reloaded.StagesValue()[1].AutoAdvanceEnabled())
	})

	t.Run("moves a workspace card to a new stage atomically", func(t *testing.T) {
		factory, line := newLine(t)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Review")
		require.NoError(t, canvas.SetColumnKey(db, "verify"))
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:2], &pb.FactoryLine_Stage{Key: "new-qa", Name: "QA"}, stages[2])
		move := &pb.FactoryLine_AutomationMove{Type: "canvas", AutomationId: canvas.ID.String(), SourceKey: "verify", DestinationKey: "new-qa"}
		config := &pb.FactoryLine_StageConfiguration{Stages: stages, AutomationMoves: []*pb.FactoryLine_AutomationMove{move, move}}
		req := &pb.UpdateFactoryLineRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), StageConfiguration: config}
		_, err := UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.Error(t, err)
		other, _ := newLine(t)
		foreign := support.CreateFactoryCanvas(t, r, other.ID, "Other")
		require.NoError(t, foreign.SetColumnKey(db, "verify"))
		config.AutomationMoves[1] = &pb.FactoryLine_AutomationMove{Type: "canvas", AutomationId: foreign.ID.String(), SourceKey: "verify", DestinationKey: "new-qa"}
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.Error(t, err)
		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		require.Len(t, reloaded.Stages, 3)
		config.AutomationMoves = config.AutomationMoves[:1]
		result, err := UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.NoError(t, err)
		card, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Equal(t, result.Line.Stages[2].Key, card.DisplayColumnKey)
		assert.Equal(t, "verify", *card.ColumnKey)
		config.Stages = []*pb.FactoryLine_Stage{result.Line.Stages[0], result.Line.Stages[1], result.Line.Stages[3]}
		move.SourceKey, move.DestinationKey = card.DisplayColumnKey, "backlog"
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.ErrorContains(t, err, "Backlog")
		card, err = models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		assert.Equal(t, result.Line.Stages[2].Key, card.DisplayColumnKey)
		assert.Equal(t, "verify", *card.ColumnKey)
	})

	t.Run("moves a task departure trigger without starting a run", func(t *testing.T) {
		factory := seedHandOffFactory(t, r, db)
		support.AssignOpenSystemTrial(t, r.Organization.ID)
		upsertHostedOnboardingProvider(t, db)
		vcs := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcs}))
		line, err := factory.CreateLine(db, "Review", nil)
		require.NoError(t, err)
		ctx, deps := onboardingContext(r), onboardingDeps(r)
		onTask := true
		created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
			FactoryId: factory.ID.String(), Name: "Check departure", ColumnKey: "backlog",
			Settings: &pb.FactoryCustomAutomation_Settings{
				StartMode: pb.FactoryCustomAutomation_START_MODE_ON_LEAVE, RunsOnTask: &onTask,
				Agent: &pb.FactoryAppTemplateAgent{Component: models.SuperPlaneRunnerComponent, Model: "hosted::anthropic::sonnet"},
				Steps: []*pb.FactoryCustomAutomation_Step{{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "pwd"}},
			},
		})
		require.NoError(t, err)
		id := created.CustomAutomation.Automation.Id
		before, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(id))
		require.NoError(t, err)
		move := &pb.FactoryLine_AutomationMove{Type: "canvas", AutomationId: id, SourceKey: "backlog", DestinationKey: "new-qa"}
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:2], &pb.FactoryLine_Stage{Key: "new-qa", Name: "QA"}, stages[2])
		req := &pb.UpdateFactoryLineRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages, AutomationMoves: []*pb.FactoryLine_AutomationMove{move}}}
		placed, err := UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), req)
		require.NoError(t, err)
		card, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(id))
		require.NoError(t, err)
		require.Equal(t, "backlog", *card.ColumnKey)
		require.Equal(t, before.LiveVersionID, card.LiveVersionID)
		require.Equal(t, placed.Line.Stages[2].Key, card.DisplayColumnKey)
		req.StageConfiguration.Stages = placed.Line.Stages
		move.SourceKey = card.DisplayColumnKey
		move.DestinationKey = "verify"
		_, err = UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), req)
		require.NoError(t, err)
		runs, err := models.ListCanvasRuns(db, uuid.MustParse(id), 10, nil, models.CanvasRunFilters{})
		require.NoError(t, err)
		require.Empty(t, runs)
		order, err := factory.CreateWorkOrder(db, "Inspect", "", &r.User, nil, nil)
		require.NoError(t, err)
		for _, stage := range []string{"backlog", "verify", "verify"} {
			events, err := contexts.EmitColumnLeaveCustomAutomations(db, factory, order, stage, "departure")
			require.NoError(t, err)
			if stage == "backlog" {
				require.Empty(t, events)
			}
		}
		runs, err = models.ListCanvasRuns(db, uuid.MustParse(id), 10, nil, models.CanvasRunFilters{})
		require.NoError(t, err)
		require.Len(t, runs, 1)
	})

	t.Run("saves incompatible task placements without changing execution", func(t *testing.T) {
		factory := seedHandOffFactory(t, r, db)
		vcs := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agent := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcs, AgentIntegrationID: &agent}))
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
		_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-opus-5-5"})
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
		step := models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint}
		line, err := factory.CreateLine(db, "Review", []models.FactoryLineStep{step, step})
		require.NoError(t, err)
		stage := models.FactoryLineStage{ID: uuid.New(), Key: "custom-qa", Name: "QA", Kind: models.FactoryLineStagePhase}
		stages := line.StagesValue()
		stages[2].Name = "Check"
		stages = append(stages[:1], append([]models.FactoryLineStage{stage}, stages[1:]...)...)
		require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
		ctx, deps := onboardingContext(r), onboardingDeps(r)
		service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
		for _, origin := range []string{stage.Key, "implement"} {
			created, err := service.Create(ctx, automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: origin}, r.User, "apiVersion: v2\nkind: Automation\nname: Inspect "+origin+"\ntriggers:\n  - use: task_enters_stage\nsteps:\n  - name: Inspect\n    bash: pwd\n")
			require.NoError(t, err)
			id := created.ID
			before, err := models.FindCanvasInTransaction(db, r.Organization.ID, id)
			require.NoError(t, err)
			source := origin
			if source == "implement" {
				source = "phase-0"
			}
			for _, destination := range []string{"verify", "phase-1"} {
				move := &pb.FactoryLine_AutomationMove{Type: "canvas", AutomationId: id.String(), ExecutionKey: origin, SourceKey: source, DestinationKey: destination}
				if origin == "implement" {
					move.ExecutionKey = "phase-0"
				}
				req := &pb.UpdateFactoryLineRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(stages), AutomationMoves: []*pb.FactoryLine_AutomationMove{move}}}
				_, err := UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), req)
				require.NoError(t, err)
				card, err := models.FindCanvasInTransaction(db, r.Organization.ID, id)
				require.NoError(t, err)
				require.Equal(t, destination, card.DisplayColumnKey)
				require.Equal(t, origin, *card.ColumnKey)
				require.Equal(t, before.LiveVersionID, card.LiveVersionID)
				sourceAfter, err := service.Read(ctx, r.Organization.ID, id)
				require.NoError(t, err)
				require.Equal(t, created.Revision, sourceAfter.Revision)
				require.Equal(t, created.YAML, sourceAfter.YAML)
				runs, err := models.ListCanvasRuns(db, id, 10, nil, models.CanvasRunFilters{})
				require.NoError(t, err)
				require.Empty(t, runs)
				if origin == stage.Key {
					req.StageConfiguration.AutomationMoves = nil
					req.StageConfiguration.Stages = factoryLineStageRequests(append([]models.FactoryLineStage{stages[0]}, stages[2:]...))
					_, err = UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), req)
					require.ErrorContains(t, err, "Keep the execution stage")
				}
				source = destination
			}
		}
		move := &pb.FactoryLine_AutomationMove{Type: "agent-step", AutomationId: app.ID.String(), ExecutionKey: "phase-0", SourceKey: "phase-0", DestinationKey: "backlog"}
		_, err = UpdateFactoryLine(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(stages), AutomationMoves: []*pb.FactoryLine_AutomationMove{move}}})
		require.ErrorContains(t, err, "Backlog")
		runs, err := models.ListCanvasRuns(db, app.ID, 10, nil, models.CanvasRunFilters{})
		require.NoError(t, err)
		require.Empty(t, runs)
	})

	t.Run("removes optional system stages", func(t *testing.T) {
		factory, line := newLine(t)
		canvas := resetTestTemplate(t, r, factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
		require.NoError(t, canvas.SetColumnKey(db, models.CanvasColumnKeyVerify))
		review := resetTestTemplate(t, r, factory, reviewRequestTemplateID, models.ProviderGitHub, func(*yaml.Canvas) {})
		feedback := support.CreateFactoryCanvas(t, r, factory.ID, "feedback")
		_, err := factory.CreatePRFeedbackHandler(db, feedback.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestChecks)
		require.NoError(t, err)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], stages[2:]...)

		updated, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Done"}, stageNames(updated.Line.Stages))
		for _, deleted := range []*models.Canvas{canvas, review, feedback} {
			_, err = models.FindCanvasInTransaction(db, r.Organization.ID, deleted.ID)
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		}
		handlers, err := factory.ListPRFeedbackHandlers(db)
		require.NoError(t, err)
		require.Empty(t, handlers)
	})

	t.Run("deletes a default phase runner and cancels unfinished work", func(t *testing.T) {
		factory, _ := newLine(t)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Implement", "start")
		retained, retainedEntrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Retained", "start")
		line, err := factory.CreateLine(db, "phases", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
			{Type: models.FactoryLineStepTypeRunApp, AppID: retained.ID, Entrypoint: retainedEntrypoint},
		})
		require.NoError(t, err)
		run, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
		require.NoError(t, err)
		finished, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
		require.NoError(t, err)
		event := support.EmitCanvasEventForNode(t, app.ID, entrypoint, "default", nil)
		require.NoError(t, db.Model(event).Update("run_id", run.ID).Error)
		queue := support.CreateQueueItem(t, app.ID, entrypoint, event.ID, event.ID)
		require.NoError(t, db.Model(queue).Update("run_id", run.ID).Error)
		execution := support.CreateCanvasNodeExecution(t, app.ID, entrypoint, event.ID, event.ID)
		require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], stages[2:]...)
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)
		_, err = models.FindCanvasInTransaction(db, r.Organization.ID, app.ID)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		require.Len(t, reloaded.Steps, 1)
		require.Equal(t, retained.ID, reloaded.Steps[0].AppID)
		require.Equal(t, stages[1].Id, reloaded.Stages[1].ID.String())
		require.Equal(t, 0, *reloaded.Stages[1].StepIndex)
		plan, err := reloaded.ExecutionPlan(db)
		require.NoError(t, err)
		require.Len(t, plan, 1)
		require.Equal(t, retained.ID, plan[0].AppID)
		updated, err := UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			Steps: serializeFactoryLine(reloaded).Steps,
		})
		require.NoError(t, err)
		require.Equal(t, stages[1].Id, updated.Line.Stages[1].Id)
		require.Equal(t, stages[1].Key, updated.Line.Stages[1].Key)
		updated, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			Steps: []*pb.FactoryLine_Step{updated.Line.Steps[0], updated.Line.Steps[0]},
		})
		require.NoError(t, err)
		require.Equal(t, stages[1].Id, updated.Line.Stages[1].Id)
		require.NotEqual(t, updated.Line.Stages[1].Key, updated.Line.Stages[2].Key)
		cancelled, err := models.FindCanvasRunInTransaction(db, app.ID, run.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateCancelling, cancelled.State)
		open, err := cancelled.FindOpenWork(db)
		require.NoError(t, err)
		assert.False(t, open.HasQueueItems)
		assert.False(t, open.HasPendingEvents)
		require.NoError(t, db.First(execution, execution.ID).Error)
		assert.Equal(t, models.CanvasNodeExecutionStateCancelling, execution.State)
		completed, err := models.FindCanvasRunInTransaction(db, app.ID, finished.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateFinished, completed.State)
		assert.Equal(t, models.CanvasRunResultPassed, completed.Result)
	})

	t.Run("keeps a runner used by a retained phase", func(t *testing.T) {
		factory, _ := newLine(t)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Repeated", "start")
		require.NoError(t, app.SetColumnKey(db, "implement"))
		step := models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint}
		line, err := factory.CreateLine(db, "repeated", []models.FactoryLineStep{step, step})
		require.NoError(t, err)
		run, err := models.CreateCanvasRunInTransaction(db, app.ID, entrypoint, models.CanvasRunStateStarted, "")
		require.NoError(t, err)
		stages := factoryLineStageRequests(line.StagesValue())
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: append(stages[:1], stages[2:]...)},
		})
		require.NoError(t, err)
		_, err = models.FindCanvasInTransaction(db, r.Organization.ID, app.ID)
		require.NoError(t, err)
		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		require.Len(t, reloaded.Steps, 1)
		assert.Equal(t, app.ID, reloaded.Steps[0].AppID)
		assert.Equal(t, "phase-1", reloaded.Stages[1].Key)
		assert.Equal(t, 0, *reloaded.Stages[1].StepIndex)
		run, err = models.FindCanvasRunInTransaction(db, app.ID, run.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateStarted, run.State)
	})

	t.Run("cancels waiting dispatches atomically when their runner is deleted", func(t *testing.T) {
		factory, _ := newLine(t)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "Queued", "start")
		one := 1
		line, err := factory.CreateLine(db, "queued", []models.FactoryLineStep{{
			Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &one,
		}})
		require.NoError(t, err)
		var waiting *models.FactoryWorkOrderLineDispatch
		for index := range 2 {
			order, err := factory.CreateWorkOrder(db, fmt.Sprintf("Order %d", index), "", &r.User, nil, nil)
			require.NoError(t, err)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				var result *models.FactoryLineStepResult
				waiting, result, err = line.Dispatch(tx, order)
				if err == nil && index == 1 {
					require.NotNil(t, result.QueueItem)
				}
				return err
			}))
		}
		stages := factoryLineStageRequests(line.StagesValue())
		req := &pb.UpdateFactoryLineRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: append(stages[:1], stages[2:]...)}}
		const callback = "test:fail_queued_stage_cleanup"
		t.Cleanup(func() { db.Callback().Update().Remove(callback) })
		require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
			if tx.Statement.Table == "factory_lines" {
				tx.AddError(errors.New("save failed"))
			}
		}))
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.Error(t, err)
		require.NoError(t, db.Callback().Update().Remove(callback))
		queued, err := models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(db, []uuid.UUID{waiting.ID})
		require.NoError(t, err)
		require.Len(t, queued, 1)
		unchanged, err := models.FindWorkOrderLineDispatch(db, waiting.ID)
		require.NoError(t, err)
		assert.True(t, unchanged.IsActive())
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.NoError(t, err)
		queued, err = models.ListFactoryWorkOrderQueueItemsByLineDispatchIDs(db, []uuid.UUID{waiting.ID})
		require.NoError(t, err)
		assert.Empty(t, queued)
		waiting, err = models.FindWorkOrderLineDispatch(db, waiting.ID)
		require.NoError(t, err)
		assert.False(t, waiting.IsActive())
		assert.Equal(t, models.CanvasRunResultCancelled, waiting.Result)
		admitted, err := models.AdmitQueuedForStep(db, line.ID, app.ID, "phase-0")
		require.NoError(t, err)
		assert.Empty(t, admitted)
	})

	t.Run("keeps Backlog and Done fixed", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())

		for _, invalid := range [][]*pb.FactoryLine_Stage{stages[1:], stages[:len(stages)-1]} {
			_, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
				FactoryId:          factory.ID.String(),
				LineId:             line.ID.String(),
				StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: invalid},
			})
			code, _, ok := grpcerrors.HandlerStatus(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, code)
		}
	})

	t.Run("reorders optional system stages", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{
			Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint,
		}})
		require.NoError(t, err)
		stages := factoryLineStageRequests(line.StagesValue())
		reordered := []*pb.FactoryLine_Stage{stages[0], stages[2], stages[1], stages[3]}

		updated, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: reordered},
		})
		require.NoError(t, err)
		require.Len(t, updated.Line.Stages, 4)
		assert.Equal(t, []string{stages[0].Id, stages[2].Id, stages[1].Id, stages[3].Id}, []string{
			updated.Line.Stages[0].Id,
			updated.Line.Stages[1].Id,
			updated.Line.Stages[2].Id,
			updated.Line.Stages[3].Id,
		})
	})

	t.Run("releases automation names when their execution stage is removed", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{
			{Name: "Security", Kind: pb.FactoryLine_STAGE_KIND_PHASE},
			{Name: "Release", Kind: pb.FactoryLine_STAGE_KIND_PHASE},
		}, stages[1:]...)...)
		created, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)

		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "security")
		originalName := canvas.Name
		removedKey, retainedKey := created.Line.Stages[1].Key, created.Line.Stages[2].Key
		require.NoError(t, canvas.SetColumnKey(db, removedKey))
		source := &models.AutomationSource{
			ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
			Name: originalName, Source: "definition", Revision: 1, ArtifactID: canvas.ID.String(),
			ArtifactRevision: canvas.LiveVersionID.String(),
		}
		require.NoError(t, source.Save(db))
		displayOnly := support.CreateFactoryCanvas(t, r, factory.ID, "display-only")
		require.NoError(t, displayOnly.SetColumnKey(db, "verify"))
		require.NoError(t, displayOnly.SetDisplayColumn(db, removedKey))
		moved := support.CreateFactoryCanvas(t, r, factory.ID, "moved")
		require.NoError(t, moved.SetColumnKey(db, removedKey))
		other, _ := newLine(t)
		foreign := support.CreateFactoryCanvas(t, r, other.ID, "foreign")
		require.NoError(t, foreign.SetColumnKey(db, removedKey))
		require.NoError(t, foreign.SetDisplayColumn(db, removedKey))
		withoutCustom := append([]*pb.FactoryLine_Stage{created.Line.Stages[0]}, created.Line.Stages[2:]...)
		_, err = UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{
				Stages: withoutCustom,
				AutomationMoves: []*pb.FactoryLine_AutomationMove{{
					Type: "canvas", AutomationId: moved.ID.String(), SourceKey: removedKey, DestinationKey: retainedKey,
				}},
			},
		})
		require.NoError(t, err)
		replacement, err := CreateFactoryAutomation(
			authentication.SetUserIdInMetadata(t.Context(), r.User.String()), deps, r.Organization.ID.String(),
			&pb.CreateFactoryAutomationRequest{FactoryId: factory.ID.String(), Name: originalName},
		)
		require.NoError(t, err)
		require.Equal(t, originalName, replacement.Automation.Name)
		_, err = models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		var deleted models.Canvas
		require.NoError(t, db.Unscoped().First(&deleted, canvas.ID).Error)
		assert.True(t, deleted.DeletedAt.Valid)
		assert.NotEqual(t, originalName, deleted.Name)
		_, err = models.FindAutomationSourceByArtifact(db, r.Organization.ID, canvas.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		found, err := models.FindCanvasByName(db, r.Organization.ID, &factory.ID, originalName)
		require.NoError(t, err)
		assert.Equal(t, replacement.Automation.Id, found.ID.String())
		for _, fixture := range []struct {
			canvas  *models.Canvas
			key     string
			display string
		}{{displayOnly, "verify", ""}, {moved, retainedKey, ""}, {foreign, removedKey, removedKey}} {
			reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, fixture.canvas.ID)
			require.NoError(t, err)
			assert.Equal(t, fixture.key, *reloaded.ColumnKey)
			assert.Equal(t, fixture.display, reloaded.DisplayColumnKey)
		}
	})

	t.Run("removes legacy shared phase bindings without removing their columns", func(t *testing.T) {
		factory, line := newLine(t)
		stages := factoryLineStageRequests(line.StagesValue())
		stages = append(stages[:1], append([]*pb.FactoryLine_Stage{{Name: "Security", Kind: pb.FactoryLine_STAGE_KIND_PHASE}}, stages[1:]...)...)
		created, err := UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: stages},
		})
		require.NoError(t, err)
		deletable := support.CreateFactoryCanvas(t, r, factory.ID, "deletable")
		protected, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "protected", "start")
		retained, retainedEntrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "retained", "start")
		other, err := factory.CreateLine(db, "other", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: protected.ID, Entrypoint: entrypoint},
			{Type: models.FactoryLineStepTypeRunApp, AppID: retained.ID, Entrypoint: retainedEntrypoint},
		})
		require.NoError(t, err)
		for _, canvas := range []*models.Canvas{deletable, protected} {
			require.NoError(t, canvas.SetColumnKey(db, created.Line.Stages[1].Key))
		}
		run, err := models.CreateCanvasRunInTransaction(db, protected.ID, entrypoint, models.CanvasRunStateStarted, "")
		require.NoError(t, err)
		req := &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(), LineId: line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{
				Stages: append([]*pb.FactoryLine_Stage{created.Line.Stages[0]}, created.Line.Stages[2:]...),
			},
		}
		const callback = "test:fail_stage_binding_cleanup"
		t.Cleanup(func() { db.Callback().Update().Remove(callback) })
		require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
			if tx.Statement.Table == "factory_lines" {
				tx.AddError(errors.New("binding cleanup failed"))
			}
		}))
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.ErrorContains(t, err, "failed to update factory line")
		require.NoError(t, db.Callback().Update().Remove(callback))
		unchanged, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, stageNames(created.Line.Stages), modelStageNames(unchanged.StagesValue()))
		for _, canvas := range []*models.Canvas{deletable, protected} {
			active, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
			require.NoError(t, err)
			assert.Equal(t, canvas.Name, active.Name)
		}
		unchangedRun, err := models.FindCanvasRunInTransaction(db, protected.ID, run.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateStarted, unchangedRun.State)
		unchangedLegacy, err := factory.FindLine(db, other.ID)
		require.NoError(t, err)
		assert.Equal(t, other.Steps, unchangedLegacy.Steps)
		assert.Equal(t, other.Stages, unchangedLegacy.Stages)
		_, err = UpdateFactoryLine(t.Context(), deps, r.Organization.ID.String(), req)
		require.NoError(t, err)
		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"Backlog", "Verify", "Done"}, modelStageNames(reloaded.StagesValue()))
		legacy, err := factory.FindLine(db, other.ID)
		require.NoError(t, err)
		require.Len(t, legacy.Steps, 1)
		assert.Equal(t, retained.ID, legacy.Steps[0].AppID)
		require.Len(t, legacy.Stages, 5)
		assert.Nil(t, legacy.Stages[1].StepIndex)
		assert.Equal(t, other.Stages[2].ID, legacy.Stages[2].ID)
		assert.Equal(t, 0, *legacy.Stages[2].StepIndex)
		plan, err := legacy.ExecutionPlan(db)
		require.NoError(t, err)
		require.Len(t, plan, 1)
		assert.Equal(t, retained.ID, plan[0].AppID)
		cancelled, err := models.FindCanvasRunInTransaction(db, protected.ID, run.ID)
		require.NoError(t, err)
		assert.Equal(t, models.CanvasRunStateCancelling, cancelled.State)
		for _, canvas := range []*models.Canvas{deletable, protected} {
			_, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		}
	})

	t.Run("updates stages without opt-in and preserves saved records", func(t *testing.T) {
		factory, line := newLine(t)
		stageID := uuid.New()
		autoAdvance := true
		stages := line.StagesValue()
		stages = append(stages[:1], append([]models.FactoryLineStage{{
			ID: stageID, Key: "custom-" + stageID.String(), Name: "QA", Kind: models.FactoryLineStagePhase, AutoAdvance: &autoAdvance,
		}}, stages[1:]...)...)
		require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "QA checks")
		require.NoError(t, canvas.SetColumnKey(db, stages[1].Key))

		_, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(line.StagesValue())},
		})
		require.NoError(t, err)
		reloaded, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, stages, reloaded.StagesValue())
		savedCanvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, savedCanvas.ColumnKey)
		assert.Equal(t, stages[1].Key, *savedCanvas.ColumnKey)
	})

	t.Run("rejects replacing steps and stages together", func(t *testing.T) {
		factory, line := newLine(t)
		_, err := UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:          factory.ID.String(),
			LineId:             line.ID.String(),
			Steps:              []*pb.FactoryLine_Step{{}},
			StageConfiguration: &pb.FactoryLine_StageConfiguration{Stages: factoryLineStageRequests(line.StagesValue())},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a step update that would exceed the stage limit", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
		step := models.FactoryLineStep{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      app.ID,
			Entrypoint: entrypoint,
		}
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{step})
		require.NoError(t, err)

		customStages := make([]models.FactoryLineStage, 60)
		for index := range customStages {
			id := uuid.New()
			customStages[index] = models.FactoryLineStage{
				ID:   id,
				Key:  "custom-" + id.String(),
				Name: fmt.Sprintf("Review %d", index+1),
				Kind: models.FactoryLineStagePhase,
			}
		}
		stages := line.StagesValue()
		stages = append(stages[:len(stages)-1], append(customStages, stages[len(stages)-1])...)
		require.NoError(t, line.UpdateConfiguration(db, nil, nil, nil, stages))

		stepRequest := &pb.FactoryLine_Step{
			Type: models.FactoryLineStepTypeRunApp,
			App:  &pb.FactoryLine_AppStep{App: app.ID.String(), Entrypoint: entrypoint},
		}
		_, err = UpdateFactoryLine(context.Background(), deps, r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			Steps:     []*pb.FactoryLine_Step{stepRequest, stepRequest},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}

func factoryLineStageRequests(stages []models.FactoryLineStage) []*pb.FactoryLine_Stage {
	return serializeFactoryLineStages(stages)
}

func stageNames(stages []*pb.FactoryLine_Stage) []string {
	result := make([]string, len(stages))
	for index, stage := range stages {
		result[index] = stage.GetName()
	}
	return result
}

func modelStageNames(stages []models.FactoryLineStage) []string {
	result := make([]string, len(stages))
	for index, stage := range stages {
		result[index] = stage.Name
	}
	return result
}
