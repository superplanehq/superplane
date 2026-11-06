package factories

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCustomAutomationScheduleRoundTripAndTransitions(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
			integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, provider)
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSProvider: &provider, VCSIntegrationID: &integrationID}))
			r.Registry.Integrations[provider] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
				Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: prFeedbackTriggerComponent(provider)})},
			})
			ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()
			settings := &pb.FactoryCustomAutomation_Settings{
				StartMode:               pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
				Schedule:                &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *", UtcOffset: 5.75},
				Agent:                   &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode", Model: "claude-sonnet-4-6"},
				ExecutionTimeoutSeconds: 7200,
				Steps: []*pb.FactoryCustomAutomation_Step{
					{Name: "Test", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make test", WorkingDirectory: "repo", Secrets: []string{"deploy-credentials"}},
					{Name: "Review", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Review the repository.", WorkingDirectory: "repo"},
				},
			}
			created, err := CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), Name: "Scheduled review", Settings: settings})
			require.NoError(t, err)
			id := created.GetCustomAutomation().GetAutomation().GetId()
			canvasID := uuid.MustParse(id)
			describe := func() *pb.FactoryCustomAutomation_Settings {
				response, err := DescribeFactoryCustomAutomation(ctx, orgID, &pb.DescribeFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: id})
				require.NoError(t, err)
				return response.GetCustomAutomation().GetSettings()
			}
			assert.True(t, proto.Equal(settings, describe()))
			_, err = ResetOrganizationBuiltInAutomations(ctx, r.Organization.ID, deps)
			require.NoError(t, err)
			assert.True(t, proto.Equal(settings, describe()), "admin reset must preserve custom schedules")
			node, err := models.FindCanvasNode(db, canvasID, customAutomationRunnerID)
			require.NoError(t, err)
			env := environmentVars(t, node.Configuration.Data()["environment"])
			assert.Equal(t, runner.WorkspaceContextRepoURL(provider, "acme/app", ""), env["REPO_URL"])
			assert.Equal(t, "main", env["BASE"])
			assert.NotContains(t, env, "PR_NUMBER")
			assert.NotContains(t, env, "PR_HEAD")
			for _, repository := range []string{"acme/app", "acme/new-app"} {
				previous := factory.OnboardingConfigValue()
				branch := "develop"
				selectedID := previous.VCSIntegrationID
				if repository != previous.AppRepository {
					selectedID = createReadyOnboardingIntegration(t, r.Organization.ID, provider)
				}
				require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
					AppRepository: &repository, BacklogRepository: &repository,
					DefaultBranch: &branch, VCSIntegrationID: &selectedID,
				}))
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					return reconcileFactoryRepository(ctx, tx, deps, factory, r.User,
						previous.VCSIntegrationID, selectedID, previous.AppRepository,
						previous.BacklogRepository, previous.DefaultBranch, repository, true)
				}))
				node, err = models.FindCanvasNode(db, canvasID, customAutomationRunnerID)
				require.NoError(t, err)
				env = environmentVars(t, node.Configuration.Data()["environment"])
				assert.Equal(t, "develop", env["BASE"])
				assert.Equal(t, repository, env["REPO"])
				assert.Equal(t, repository, env["SOURCE_REPO"])
				assert.Equal(t, runner.WorkspaceContextRepoURL(provider, repository, ""), env["REPO_URL"])
				assert.NotContains(t, env, "PR_HEAD")
				assert.Equal(t, integrationName(t, r.Organization.ID, selectedID),
					node.Configuration.Data()["environmentFrom"].([]any)[0].(map[string]any)["integration"].(map[string]any)["name"])
				assert.True(t, proto.Equal(settings, describe()), "repository updates must preserve automation settings")
			}
			for _, invalid := range []*pb.FactoryCustomAutomation_Schedule{
				nil, {}, {CronExpression: "bad"}, {CronExpression: "0 9 31 2 *"},
				{CronExpression: "0 9 * * *", UtcOffset: 15}, {CronExpression: "0 9 * * *", UtcOffset: -13},
				{CronExpression: "0 9 * * *", UtcOffset: math.NaN()}, {CronExpression: "0 9 * * *", UtcOffset: math.Inf(1)},
			} {
				broken := proto.Clone(settings).(*pb.FactoryCustomAutomation_Settings)
				broken.Schedule = invalid
				_, err := UpdateFactoryCustomAutomation(ctx, deps, orgID, &pb.UpdateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: id, Name: "Scheduled review", Settings: broken})
				code, _, ok := grpcerrors.HandlerStatus(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, code)
				assert.True(t, proto.Equal(settings, describe()), "invalid update must not publish")
				_, err = CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), Name: "Invalid", Settings: broken})
				code, _, ok = grpcerrors.HandlerStatus(err)
				require.True(t, ok)
				require.Equal(t, codes.InvalidArgument, code)
			}
			for _, mode := range []pb.FactoryCustomAutomation_StartMode{pb.FactoryCustomAutomation_START_MODE_SCHEDULED, pb.FactoryCustomAutomation_START_MODE_MANUAL, pb.FactoryCustomAutomation_START_MODE_AUTOMATIC, pb.FactoryCustomAutomation_START_MODE_SCHEDULED} {
				settings.StartMode = mode
				settings.Schedule = nil
				if mode == pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
					settings.Schedule = &pb.FactoryCustomAutomation_Schedule{CronExpression: "30 0 9 * * MON-FRI"}
				}
				_, err := UpdateFactoryCustomAutomation(ctx, deps, orgID, &pb.UpdateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: id, Name: "Scheduled review", Settings: settings})
				require.NoError(t, err)
				assert.True(t, proto.Equal(settings, describe()))
			}
		})
	}
}

func TestCustomAutomationSettingsAllowBashOnly(t *testing.T) {
	r := support.Setup(t)
	factory := seedHandOffFactory(t, r, database.DB(t.Context()))

	settings, err := parseCustomAutomationSettings(database.DB(t.Context()), factory, &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Lint", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make lint"},
		},
	})
	require.NoError(t, err)
	require.Len(t, settings.Steps, 1)
	assert.Equal(t, runner.AgentStepBash, settings.Steps[0].Type)
	assert.Equal(t, prFeedbackTimeoutSeconds, settings.ExecutionTimeoutSeconds)
}

func TestBacklogCustomAutomationLifecycle(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Inspect the task.", WorkingDirectory: "repo"},
			{Name: "Check", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "true"},
		},
	}
	_, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), ColumnKey: "done", Settings: settings,
	})
	require.Error(t, err)
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Inspect task", ColumnKey: "backlog", Settings: settings,
	})
	require.NoError(t, err)
	automationID := created.GetCustomAutomation().GetAutomation().GetId()
	assert.Equal(t, "backlog", created.GetCustomAutomation().GetAutomation().GetColumnKey())
	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(automationID))
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	trigger := customAutomationNode([]models.Node(version.Nodes), customAutomationTaskTriggerID)
	require.NotNil(t, trigger)
	assert.Equal(t, "onWorkOrder", trigger.ComponentName())
	for _, node := range version.Nodes {
		assert.NotEqual(t, "github.onPullRequest", node.ComponentName())
	}
	order, err := factory.CreateWorkOrder(db, "Inspect this task", "Task context", &r.User, nil, nil)
	require.NoError(t, err)
	contexts.EmitWorkOrderCreated(db, factory, order)
	runs, err := order.ListRuns(db)
	require.NoError(t, err)
	require.Empty(t, runs)
	started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, started.GetRunId())
	event, err := models.FindCanvasEventForCanvas(db, canvas.ID, uuid.MustParse(started.GetEventId()))
	require.NoError(t, err)
	payload := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	assert.Equal(t, order.ID.String(), payload["workOrder"].(map[string]any)["id"])
	runs, err = order.ListRuns(db)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	labels, err := models.ResolveCanvasRunTasks(db, canvas, runs)
	require.NoError(t, err)
	assert.Equal(t, order.ID, labels[runs[0].ID].ID)
	settings.StartMode = pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
	settings.Steps[1].Command = "make lint"
	updated, err := UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "Inspect task", Settings: settings,
	})
	require.NoError(t, err)
	assert.Equal(t, "backlog", updated.GetCustomAutomation().GetAutomation().GetColumnKey())
	assert.Equal(t, "make lint", updated.GetCustomAutomation().GetSettings().GetSteps()[1].GetCommand())
	runs, err = order.ListRuns(db)
	require.NoError(t, err)
	require.Len(t, runs, 1, "saving must not start a run")
	second, err := factory.CreateWorkOrder(db, "New task", "Automatic context", &r.User, nil, nil)
	require.NoError(t, err)
	contexts.EmitWorkOrderCreated(db, factory, second)
	contexts.EmitWorkOrderCreated(db, factory, second)
	runs, err = second.ListRuns(db)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	_, err = second.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultArchived, Actor: &r.User})
	require.NoError(t, err)
	_, err = SendWorkOrderToBacklog(ctx, deps, r.Organization.ID.String(), &pb.SendWorkOrderToBacklogRequest{FactoryId: factory.ID.String(), OrderId: second.ID.String()})
	require.NoError(t, err)
	second, err = factory.FindWorkOrder(db, second.ID)
	require.NoError(t, err)
	contexts.EmitBacklogCustomAutomations(db, factory, second)
	runs, err = second.ListRuns(db)
	require.NoError(t, err)
	assert.Len(t, runs, 2)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, second.State)
	for i, state := range []pb.WorkOrder_State{pb.WorkOrder_STATE_OPEN, pb.WorkOrder_STATE_CLOSED} {
		_, err = UpdateWorkOrderStatus(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factory.ID.String(), OrderId: second.ID.String(), State: state, Result: pb.WorkOrder_RESULT_ARCHIVED,
		})
		require.NoError(t, err)
		_, err = UpdateWorkOrderStatus(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factory.ID.String(), OrderId: second.ID.String(), State: pb.WorkOrder_STATE_DRAFT,
		})
		require.NoError(t, err)
		second, err = factory.FindWorkOrder(db, second.ID)
		require.NoError(t, err)
		runs, err = second.ListRuns(db)
		require.NoError(t, err)
		expectedRuns := 3 + i
		require.Len(t, runs, expectedRuns, "the status API must start a run for each Backlog entry")
		contexts.EmitBacklogCustomAutomations(db, factory, second)
		runs, err = second.ListRuns(db)
		require.NoError(t, err)
		assert.Len(t, runs, expectedRuns, "repeated delivery must not start another run")
	}
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultArchived, Actor: &r.User})
	require.NoError(t, err)
	settings.StartMode = pb.FactoryCustomAutomation_START_MODE_MANUAL
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "Inspect task", Settings: settings,
	})
	require.NoError(t, err)
	_, err = RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: order.ID.String(),
	})
	require.Error(t, err, "closed tasks are not eligible")
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "Other factory", "", "")
	require.NoError(t, err)
	otherOrder, err := otherFactory.CreateWorkOrder(db, "Other task", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: otherOrder.ID.String(),
	})
	require.Error(t, err, "tasks from another factory are not eligible")
}

func TestBacklogScheduledCustomAutomationCreatesATask(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	prompt := "Look up broken ESLint rules and create a task for the simplest one."
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
		Schedule:  &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *", UtcOffset: 1},
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Find a rule", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: prompt, WorkingDirectory: "repo"},
		},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "ESLint tasks", ColumnKey: "backlog", Settings: settings,
	})
	require.NoError(t, err)
	described := created.GetCustomAutomation().GetSettings()
	require.Len(t, described.GetSteps(), 1)
	assert.Equal(t, prompt, described.GetSteps()[0].GetPrompt())
	assert.NotContains(t, described.GetSteps()[0].GetPrompt(), "/tmp/proposal.json")

	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId()))
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	trigger := findComponentNode(t, version.Nodes, "schedule")
	assert.Equal(t, "cron", trigger.Configuration["type"])
	now := time.Now()
	require.NoError(t, db.Create(&models.CanvasNode{
		WorkflowID: canvas.ID, NodeID: "schedule-orphan", Name: "Schedule", Type: models.NodeTypeTrigger,
		State: models.CanvasNodeStateReady, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}), Metadata: datatypes.NewJSONType(map[string]any{}),
		Position: datatypes.NewJSONType(models.Position{}), CreatedAt: &now, UpdatedAt: &now,
	}).Error)
	started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: canvas.ID.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, started.GetRunId())
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, uuid.MustParse(started.GetRunId()))
	require.NoError(t, err)
	assert.Equal(t, trigger.ID, run.NodeID)
	assert.Equal(t, "0 9 * * *", trigger.Configuration["cronExpression"])
	agent := findComponentNode(t, version.Nodes, "runnerClaudeCode")
	assert.Equal(t, backlogTaskRunnerName, agent.Name)
	env := environmentVars(t, agent.Configuration["environment"])
	assert.Equal(t, "main", env["BASE"])
	assert.NotContains(t, env, "PR_NUMBER")
	steps, _ := agent.Configuration["steps"].([]any)
	require.Len(t, steps, 1)
	publishedPrompt, _ := steps[0].(map[string]any)["prompt"].(string)
	assert.Contains(t, publishedPrompt, prompt)
	assert.Contains(t, publishedPrompt, "create_task")
	assert.NotContains(t, publishedPrompt, "/tmp/proposal.json")
	for _, node := range version.Nodes {
		assert.NotEqual(t, "if", node.ComponentName())
		assert.NotEqual(t, "createWorkOrder", node.ComponentName())
		assert.NotEqual(t, "addRunError", node.ComponentName())
		assert.NotEqual(t, "onWorkOrder", node.ComponentName())
	}
	for _, edge := range version.Edges {
		assert.Equal(t, customAutomationRunnerID, edge.TargetID)
	}

	settings.Steps[0].Prompt = "Find the next ESLint rule."
	updated, err := UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: canvas.ID.String(), Name: "ESLint tasks", Settings: settings,
	})
	require.NoError(t, err)
	assert.Equal(t, "Find the next ESLint rule.", updated.GetCustomAutomation().GetSettings().GetSteps()[0].GetPrompt())
	version, err = models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	agent = findComponentNode(t, version.Nodes, "runnerClaudeCode")
	steps, _ = agent.Configuration["steps"].([]any)
	require.Len(t, steps, 1)
	publishedPrompt, _ = steps[0].(map[string]any)["prompt"].(string)
	assert.Equal(t, 1, strings.Count(publishedPrompt, "call the create_task tool."))
}

func TestCustomAutomationMultipleTriggers(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	settings := &pb.FactoryCustomAutomation_Settings{
		Agent: &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Find a rule", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Find a task.", WorkingDirectory: "repo"},
		},
		Triggers: []*pb.FactoryCustomAutomation_StartTrigger{
			{StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED, Schedule: &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *", UtcOffset: 1}},
			{StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC},
			{StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED, Schedule: &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 18 * * *", UtcOffset: 1}},
		},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "ESLint tasks", ColumnKey: "backlog", Settings: settings,
	})
	require.NoError(t, err)
	automationID := created.GetCustomAutomation().GetAutomation().GetId()
	assertCustomAutomationTriggers(t, created.GetCustomAutomation().GetSettings(), settings.GetTriggers())
	assert.Equal(t, "Find a task.", created.GetCustomAutomation().GetSettings().GetSteps()[0].GetPrompt())
	assert.NotContains(t, created.GetCustomAutomation().GetSettings().GetSteps()[0].GetPrompt(), "/tmp/proposal.json")

	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(automationID))
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	schedules := 0
	var runner *models.Node
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if node.ComponentName() == "schedule" {
			schedules++
		}
		if node.ComponentName() != "runnerClaudeCode" {
			continue
		}
		require.Nil(t, runner)
		runner = node
	}
	assert.Equal(t, 2, schedules)
	require.NotNil(t, runner)
	assert.Equal(t, backlogTaskRunnerName, runner.Name)
	assert.Equal(t, customAutomationRunnerID, runner.ID)
	env := environmentVars(t, runner.Configuration["environment"])
	assert.Contains(t, env["REPO"], "workOrder?.repository")
	assert.Contains(t, env["BASE"], "main")
	assert.NotContains(t, env, "PR_NUMBER")
	steps, _ := runner.Configuration["steps"].([]any)
	require.Len(t, steps, 1)
	assert.Contains(t, steps[0].(map[string]any)["prompt"], "create_task")
	assert.NotContains(t, steps[0].(map[string]any)["prompt"], "/tmp/proposal.json")
	for _, node := range version.Nodes {
		assert.NotEqual(t, "if", node.ComponentName())
		assert.NotEqual(t, "createWorkOrder", node.ComponentName())
		assert.NotEqual(t, "addRunError", node.ComponentName())
	}
	for _, edge := range version.Edges {
		assert.Equal(t, runner.ID, edge.TargetID)
	}
	trigger := customAutomationNode([]models.Node(version.Nodes), customAutomationTaskTriggerID)
	require.NotNil(t, trigger)
	template, _ := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	assert.Equal(t, "automatic", template["startMode"])

	order, err := factory.CreateWorkOrder(db, "Inspect this task", "Task context", &r.User, nil, nil)
	require.NoError(t, err)
	contexts.EmitWorkOrderCreated(db, factory, order)
	runs, err := order.ListRuns(db)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	_, err = RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: order.ID.String(),
	})
	require.Error(t, err, "a task event does not allow a manual start")

	settings.Triggers = append([]*pb.FactoryCustomAutomation_StartTrigger{
		{StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL},
	}, settings.GetTriggers()...)
	updated, err := UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "ESLint tasks", Settings: settings,
	})
	require.NoError(t, err)
	assertCustomAutomationTriggers(t, updated.GetCustomAutomation().GetSettings(), settings.GetTriggers())

	canvas, err = models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	version, err = models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	renamed, err := UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "ESLint Error Fixer", Settings: settings,
	})
	require.NoError(t, err)
	assert.Equal(t, "ESLint Error Fixer", renamed.GetCustomAutomation().GetAutomation().GetName())
	assertCustomAutomationTriggers(t, renamed.GetCustomAutomation().GetSettings(), settings.GetTriggers())
	started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, started.GetRunId())

	duplicate := proto.Clone(settings).(*pb.FactoryCustomAutomation_Settings)
	duplicate.Triggers = append(duplicate.Triggers, &pb.FactoryCustomAutomation_StartTrigger{StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC})
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "ESLint tasks", Settings: duplicate,
	})
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, code)
	described, err := DescribeFactoryCustomAutomation(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID,
	})
	require.NoError(t, err)
	assertCustomAutomationTriggers(t, described.GetCustomAutomation().GetSettings(), settings.GetTriggers())

	_, err = ResetOrganizationBuiltInAutomations(ctx, r.Organization.ID, deps)
	require.NoError(t, err)
	described, err = DescribeFactoryCustomAutomation(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID,
	})
	require.NoError(t, err)
	assertCustomAutomationTriggers(t, described.GetCustomAutomation().GetSettings(), settings.GetTriggers())

	previous := factory.OnboardingConfigValue()
	repository := "acme/new-app"
	branch := "develop"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		AppRepository: &repository, BacklogRepository: &repository, DefaultBranch: &branch,
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return reconcileFactoryRepository(ctx, tx, deps, factory, r.User,
			previous.VCSIntegrationID, previous.VCSIntegrationID, previous.AppRepository,
			previous.BacklogRepository, previous.DefaultBranch, repository, true)
	}))
	canvas, err = models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	version, err = models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	runner = nil
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if node.ComponentName() != "runnerClaudeCode" {
			continue
		}
		require.Nil(t, runner)
		runner = node
	}
	require.NotNil(t, runner)
	env = environmentVars(t, runner.Configuration["environment"])
	assert.Contains(t, env["REPO"], "workOrder?.repository")
	assert.Contains(t, env["REPO"], repository)
	assert.Contains(t, env["BASE"], branch)
}

func assertCustomAutomationTriggers(t *testing.T, settings *pb.FactoryCustomAutomation_Settings, want []*pb.FactoryCustomAutomation_StartTrigger) {
	t.Helper()
	require.Len(t, settings.GetTriggers(), len(want))
	for i, trigger := range want {
		assert.Equal(t, trigger.GetStartMode(), settings.GetTriggers()[i].GetStartMode())
		assert.Equal(t, trigger.GetSchedule().GetCronExpression(), settings.GetTriggers()[i].GetSchedule().GetCronExpression())
		assert.Equal(t, trigger.GetSchedule().GetUtcOffset(), settings.GetTriggers()[i].GetSchedule().GetUtcOffset())
	}
}

func findComponentNode(t *testing.T, nodes []models.Node, name string) *models.Node {
	t.Helper()
	for i := range nodes {
		if nodes[i].ComponentName() == name {
			return &nodes[i]
		}
	}
	t.Fatalf("component %s not found", name)
	return nil
}

func TestVerifyCustomAutomationRunsOncePerWorkspaceRepositoryPullRequest(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	db := database.DB(t.Context())
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "review", "start-review")
	_, err = factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	})
	require.NoError(t, err)
	api := openHandOffPullRequest("acme/app", 42, "feat/api")
	api.Head.SHA = github.Ptr("api-sha")
	web := openHandOffPullRequest("acme/web", 7, "feat/web")
	web.Head.SHA = github.Ptr("web-sha")
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return &fakeFactoryGitHub{pullRequests: map[string]*github.PullRequest{"acme/app#42": api, "acme/web#7": web}}, nil
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
	handOff, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID:       factory.ID.String(),
		Title:           "Ship refunds in API and web",
		Plan:            "# Plan",
		Column:          HandOffColumnVerify,
		PullRequestURL:  "https://github.com/acme/app/pull/42",
		PullRequestURLs: []string{"https://github.com/acme/web/pull/7"},
	})
	require.NoError(t, err)
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Verify changes", ColumnKey: "verify",
		Settings: &pb.FactoryCustomAutomation_Settings{
			StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
			Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
			Steps: []*pb.FactoryCustomAutomation_Step{
				{Name: "Test", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make test", WorkingDirectory: "repo"},
			},
		},
	})
	require.NoError(t, err)
	automationID := created.GetCustomAutomation().GetAutomation().GetId()

	_, err = RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: handOff.Order.GetId(),
	})
	require.NoError(t, err)

	startedRepositories := func() []string {
		var events []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ?", automationID).Find(&events).Error)
		repositories := make([]string, 0, len(events))
		for _, event := range events {
			payload := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
			repositories = append(repositories, payload["repository"].(map[string]any)["full_name"].(string))
		}
		return repositories
	}
	assert.ElementsMatch(t, []string{"acme/app", "acme/web"}, startedRepositories(), "each pull request must start a run on its own repository")

	trigger, err := models.FindCanvasNode(db, uuid.MustParse(automationID), customAutomationTriggerID)
	require.NoError(t, err)
	configuration := trigger.Configuration.Data()
	configuration["repository"] = "acme/app"
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", trigger.WorkflowID, trigger.NodeID).
		Update("configuration", datatypes.NewJSONType(configuration)).Error)
	_, err = RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, WorkOrderId: handOff.Order.GetId(),
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"acme/app", "acme/web", "acme/app"}, startedRepositories(), "an automation saved for one repository must not check out another repository's pull request")
}

func TestCustomAutomationGraphUsesProviderAndFocusedSettings(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			integrationID := uuid.NewString()
			branch := "develop"
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
				VCSProvider:      &provider,
				VCSIntegrationID: &integrationID,
				DefaultBranch:    &branch,
			}))
			prompt := "Review the pull request."
			command := "make test"
			settings, err := parseCustomAutomationSettings(db, factory, &pb.FactoryCustomAutomation_Settings{
				StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC,
				Agent: &pb.FactoryAppTemplateAgent{
					Component:                 "runnerClaudeCode",
					Model:                     "claude-sonnet-4-6",
					CredentialSource:          runner.CredentialsSourceIntegration,
					CredentialIntegrationName: "Claude",
				},
				ExecutionTimeoutSeconds: 7200,
				Steps: []*pb.FactoryCustomAutomation_Step{
					{Name: "Review", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: prompt, WorkingDirectory: "repo"},
					{
						Name:             "Test",
						Type:             pb.FactoryCustomAutomation_Step_TYPE_BASH,
						Command:          command,
						WorkingDirectory: "repo",
						Secrets:          []string{"deploy-credentials"},
					},
				},
			})
			require.NoError(t, err)

			doc, err := buildCustomAutomationCanvas(db, factory, "Verify changes", settings)
			require.NoError(t, err)
			require.Len(t, doc.Spec.Nodes, 2)
			trigger := doc.Spec.Nodes[0]
			agent := doc.Spec.Nodes[1]

			assert.Equal(t, prFeedbackTriggerComponent(provider), trigger.Component)
			assert.Equal(t, prFeedbackTriggerActions(provider), trigger.Configuration["actions"])
			assert.Equal(t, integrationID, trigger.Integration.ID)
			template := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
			assert.Equal(t, models.FactoryAppTemplateCustomAutomationID, template["id"])
			assert.Equal(t, "automatic", template["startMode"])
			assert.Equal(t, "runnerClaudeCode", agent.Component)
			assert.Equal(t, "claude-sonnet-4-6", agent.Configuration["model"])
			assert.Equal(t, 7200, agent.Configuration["executionTimeoutSeconds"])
			assert.Equal(t, customAutomationStepConfiguration(settings.Steps), agent.Configuration["steps"])
			environmentFrom := agent.Configuration["environmentFrom"].([]any)
			require.Len(t, environmentFrom, 2)
			secretSource := environmentFrom[1].(map[string]any)
			assert.Equal(t, runner.EnvironmentFromSourceSecret, secretSource["source"])
			assert.Equal(t, "deploy-credentials", secretSource["secret"].(map[string]any)["secret"])
			got := environmentVars(t, agent.Configuration["environment"])
			if provider == models.ProviderGitHub {
				assert.Equal(t, models.WorkspaceRepositoriesExpression, trigger.Configuration["repository"])
				assert.Equal(t, prFeedbackEventRepositoryExpression, got["REPO"], "the run must check out the repository of its pull request")
				assert.Equal(t, prFeedbackEventRepositoryExpression, got["SOURCE_REPO"])
				assert.Equal(t, runner.WorkspaceContextRepoURL(provider, prFeedbackEventRepositoryExpression, ""), got["REPO_URL"])
				assert.NotContains(t, got, "BASE")
			} else {
				assert.Equal(t, "acme/app", trigger.Configuration["repository"])
				assert.Equal(t, "acme/app", got["REPO"])
				assert.Equal(t, "acme/app", got["SOURCE_REPO"])
				assert.Equal(t, runner.WorkspaceContextRepoURL(provider, "acme/app", ""), got["REPO_URL"])
				assert.Equal(t, "develop", got["BASE"])
			}
			assert.Equal(t, prFeedbackPRNumberExpression(), got["PR_NUMBER"])
			assert.Equal(t, prFeedbackPRHeadExpression(), got["PR_HEAD"])
			_, err = structpb.NewStruct(agent.Configuration)
			require.NoError(t, err)
		})
	}
}

func TestCustomAutomationOmitsBaseWhenWorkspaceBranchIsEmpty(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	integrationID := uuid.NewString()
	empty := ""
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
		DefaultBranch:    &empty,
	}))

	settings, err := parseCustomAutomationSettings(db, factory, &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Lint", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make lint"},
		},
	})
	require.NoError(t, err)
	doc, err := buildCustomAutomationCanvas(db, factory, "Verify changes", settings)
	require.NoError(t, err)

	_, hasBase := environmentVars(t, doc.Spec.Nodes[1].Configuration["environment"])["BASE"]
	assert.False(t, hasBase)
}

func TestCustomAutomationPreservesSubscriptionAgentOnUpdate(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	subscription := true
	harness := models.FactoryOnboardingAgentHarnessCodex
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID:  &integrationID,
		AgentHarness:      &harness,
		CodexSubscription: &subscription,
	}))
	// GitHub setup is external to saving the agent configuration.
	r.Registry.Integrations["github"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: "github.onPullRequest"})},
	})
	ctx := onboardingContext(r)
	deps := onboardingDeps(r)
	organizationID := r.Organization.ID.String()
	created, err := CreateFactoryCustomAutomation(ctx, deps, organizationID, &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(),
		Name:      "Review changes",
		Settings: &pb.FactoryCustomAutomation_Settings{
			StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
			Steps:     []*pb.FactoryCustomAutomation_Step{{Name: "Review", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Review the pull request."}},
		},
	})
	require.NoError(t, err)
	automationID := created.GetCustomAutomation().GetAutomation().GetId()
	described, err := DescribeFactoryCustomAutomation(ctx, organizationID, &pb.DescribeFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID,
	})
	require.NoError(t, err)
	settings := described.GetCustomAutomation().GetSettings()
	require.Equal(t, "runnerCodex", settings.GetAgent().GetComponent())
	require.Equal(t, runner.CredentialsSourceSubscription, settings.GetAgent().GetCredentialSource())
	settings.Steps[0].Prompt = "Review the changed files."
	_, err = UpdateFactoryCustomAutomation(ctx, deps, organizationID, &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID, Name: "Review changed files", Settings: settings,
	})
	require.NoError(t, err)
	saved, err := DescribeFactoryCustomAutomation(ctx, organizationID, &pb.DescribeFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: automationID,
	})
	require.NoError(t, err)
	assert.Equal(t, "Review changed files", saved.GetCustomAutomation().GetAutomation().GetName())
	assert.Equal(t, "Review the changed files.", saved.GetCustomAutomation().GetSettings().GetSteps()[0].GetPrompt())
	assert.Equal(t, "runnerCodex", saved.GetCustomAutomation().GetSettings().GetAgent().GetComponent())
	assert.Equal(t, runner.CredentialsSourceSubscription, saved.GetCustomAutomation().GetSettings().GetAgent().GetCredentialSource())
}

func TestCustomAutomationUsesTheWorkspaceAgentWhenTheHarnessDiffers(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	customID := createReadyOnboardingIntegration(t, r.Organization.ID, models.CustomLLMAppName)
	claudeID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID:   &vcsID,
		AgentIntegrationID: &customID,
	}))
	custom, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(customID))
	require.NoError(t, err)
	claude, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(claudeID))
	require.NoError(t, err)
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
		Schedule:  &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *", UtcOffset: 1},
		Agent: &pb.FactoryAppTemplateAgent{
			Component:                 "runnerClaudeCode",
			Model:                     "claude-sonnet-5",
			CredentialSource:          runner.CredentialsSourceIntegration,
			CredentialIntegrationName: claude.InstallationName,
		},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Find a rule", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Find a task.", WorkingDirectory: "repo"},
		},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "ESLint tasks", ColumnKey: models.CanvasColumnKeyBacklog, Settings: settings,
	})
	require.NoError(t, err)
	agent := created.GetCustomAutomation().GetSettings().GetAgent()
	assert.Equal(t, "runnerClaudeCode", agent.GetComponent())
	assert.Equal(t, "claude-sonnet-5", agent.GetModel())
	assert.Equal(t, claude.InstallationName, agent.GetCredentialIntegrationName())
	assert.NotEqual(t, custom.InstallationName, agent.GetCredentialIntegrationName())
}

func TestCustomAutomationSuperPlaneAgentOmitsCredentials(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID,
	}))
	r.Registry.Integrations["github"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: "github.onPullRequest"})},
	})
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	anthropic := models.UsageProviderAnthropic
	modelName := "claude-sonnet-4-6"
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      anthropic,
		Enabled:       true,
		APIKey:        []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{modelName},
	})
	require.NoError(t, err)
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		DefaultHostedProvider: &anthropic,
		DefaultHostedModel:    &modelName,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{})
		_ = db.Where("provider = ?", anthropic).Delete(&models.HostedLLMProvider{}).Error
	})
	ctx := onboardingContext(r)
	deps := onboardingDeps(r)
	organizationID := r.Organization.ID.String()
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC,
		Agent: &pb.FactoryAppTemplateAgent{
			Component:        models.SuperPlaneRunnerComponent,
			CredentialSource: runner.CredentialsSourceHosted,
		},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Deploy", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "true", WorkingDirectory: "repo"},
		},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, organizationID, &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(),
		Name:      "Preview Environment",
		Settings:  settings,
	})
	require.NoError(t, err)
	automationID := created.GetCustomAutomation().GetAutomation().GetId()
	canvasID := uuid.MustParse(automationID)

	node := requireSuperPlaneAutomationNode(t, db, canvasID)
	assert.Equal(t, models.CanvasNodeStateReady, node.State)
	assert.Nil(t, node.StateReason)
	assert.NotContains(t, node.Configuration.Data(), "credentials")

	reason := "credentials is not supported on Run SuperPlane Agent"
	broken := node.Configuration.Data()
	broken["credentials"] = map[string]any{"source": runner.CredentialsSourceHosted}
	node.Configuration = datatypes.NewJSONType(broken)
	node.State = models.CanvasNodeStateError
	node.StateReason = &reason
	require.NoError(t, db.Save(node).Error)

	_, err = UpdateFactoryCustomAutomation(ctx, deps, organizationID, &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId:    factory.ID.String(),
		AutomationId: automationID,
		Name:         "Preview Environment",
		Settings:     settings,
	})
	require.NoError(t, err)

	repaired := requireSuperPlaneAutomationNode(t, db, canvasID)
	assert.Equal(t, models.CanvasNodeStateReady, repaired.State)
	assert.Nil(t, repaired.StateReason)
	assert.NotContains(t, repaired.Configuration.Data(), "credentials")
}

func TestTaskCustomAutomationTriggerSkipsTheSchedule(t *testing.T) {
	version := &models.CanvasVersion{Nodes: []models.Node{
		{ID: "schedule-1", Type: models.NodeTypeTrigger, Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}},
		{ID: "on-pull-request", Type: models.NodeTypeTrigger, Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}}},
	}}
	assert.Equal(t, "on-pull-request", taskCustomAutomationTriggerID(version))
}

func TestCustomAutomationKeepsAUserStepNamedWriteTaskResult(t *testing.T) {
	for _, column := range []string{models.CanvasColumnKeyVerify, models.CanvasColumnKeyBacklog} {
		t.Run(column, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
			integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
			ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()
			settings := &pb.FactoryCustomAutomation_Settings{
				StartMode: pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
				Schedule:  &pb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *"},
				Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
				Steps: []*pb.FactoryCustomAutomation_Step{
					{Name: "Write task result", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Write the task result in the pull request."},
					{Name: "Write task result", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "cat /tmp/proposal.json", WorkingDirectory: "repo"},
				},
			}
			created, err := CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), Name: "Task results", ColumnKey: column, Settings: settings,
			})
			require.NoError(t, err)
			id := created.GetCustomAutomation().GetAutomation().GetId()
			described, err := DescribeFactoryCustomAutomation(ctx, orgID, &pb.DescribeFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: id})
			require.NoError(t, err)
			require.Equal(t, settings.GetSteps(), described.GetCustomAutomation().GetSettings().GetSteps())
			_, err = UpdateFactoryCustomAutomation(ctx, deps, orgID, &pb.UpdateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), AutomationId: id, Name: "Task results", Settings: described.GetCustomAutomation().GetSettings(),
			})
			require.NoError(t, err)
			canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(id))
			require.NoError(t, err)
			version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
			require.NoError(t, err)
			agent := findComponentNode(t, version.Nodes, "runnerClaudeCode")
			steps := agent.Configuration["steps"].([]any)
			require.Len(t, steps, 2)
			assert.Equal(t, "cat /tmp/proposal.json", steps[1].(map[string]any)["command"])
		})
	}
}

func customAutomationNode(nodes []models.Node, id string) *models.Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func requireSuperPlaneAutomationNode(t *testing.T, db *gorm.DB, canvasID uuid.UUID) *models.CanvasNode {
	t.Helper()
	node, err := models.FindCanvasNode(db, canvasID, customAutomationRunnerID)
	require.NoError(t, err)
	assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
	return node
}

func environmentVars(t *testing.T, raw any) map[string]string {
	t.Helper()
	entries, ok := raw.([]any)
	require.True(t, ok)
	got := make(map[string]string, len(entries))
	for _, entry := range entries {
		item, ok := entry.(map[string]any)
		require.True(t, ok)
		name, _ := item["name"].(string)
		value, _ := item["value"].(string)
		require.NotEmpty(t, name)
		got[name] = value
	}
	return got
}
