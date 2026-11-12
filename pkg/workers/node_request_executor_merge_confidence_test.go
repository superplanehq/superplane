package workers

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestNodeRequestExecutorSkipsMergeConfidenceForUnrelatedPolls(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("poll"), "", "")
	require.NoError(t, err)
	componentName := "unrelated_poll_" + uuid.NewString()
	r.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName, Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			return ctx.Requests.ScheduleActionCall("poll", nil, time.Minute)
		},
	})
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "unrelated", Type: models.NodeTypeComponent,
		Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
	root := support.EmitCanvasEventForNode(t, canvas.ID, "unrelated", "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "unrelated", root.ID, root.ID, map[string]any{})
	require.Empty(t, execution.FrozenComponentName())
	require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	require.NoError(t, contexts.NewExecutionRequestContext(db, execution).ScheduleActionCall("poll", nil, time.Minute))

	reads := map[string]int{}
	queries := db.Callback().Query()
	name := "test:unrelated-poll:" + uuid.NewString()
	require.NoError(t, queries.Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if slices.Contains([]string{"workflows", "workflow_runs", "workflow_versions"}, tx.Statement.Table) {
			reads[tx.Statement.Table]++
		}
	}))
	defer func() { require.NoError(t, queries.Remove(name)) }()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	var previousID uuid.UUID
	for range 3 {
		var pending []models.CanvasNodeRequest
		require.NoError(t, db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Find(&pending).Error)
		require.Len(t, pending, 1, "unfinished polls must keep one scheduled request")
		request := pending[0]
		assert.NotEqual(t, previousID, request.ID)
		assert.Equal(t, "poll", request.Spec.Data().InvokeAction.ActionName)
		assert.True(t, request.RunAt.After(time.Now()))
		clear(reads)
		require.NoError(t, executor.RunNow(request))
		// The hook needs its canvas; Merge Confidence must add no canvas, run, or version reads.
		assert.Equal(t, map[string]int{"workflows": 1}, reads)
		previousID = request.ID
	}
	var pending int64
	require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Count(&pending).Error)
	assert.EqualValues(t, 1, pending)
	require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateStarted, execution.State)
	var completions int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("execution_id = ?", execution.ID).Count(&completions).Error)
	assert.Zero(t, completions)
}

func TestNodeRequestExecutorPreparesMergeConfidenceOnlyOnTerminalPoll(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed", "lookup failure"} {
		t.Run(outcome, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("poll"), "", "")
			require.NoError(t, err)
			order, err := workspace.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
			require.NoError(t, err)
			configuration := map[string]any{
				"environment": []any{map[string]any{"name": runner.EnvSuperplaneMergeConfidenceOrderID, "value": order.ID.String()}},
				"steps": []any{
					map[string]any{"name": "Blast radius", "prompt": "Merge check: risk."},
					map[string]any{"name": "Security", "prompt": "Merge check: security."},
				},
			}
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: "trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onPullRequest"}})},
				{NodeID: "agent", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}), Configuration: datatypes.NewJSONType(configuration)},
			}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
			root := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger", "default", nil, map[string]any{
				"pull_request": map[string]any{"head": map[string]any{"sha": "reviewed-head"}},
			})
			_, err = pr.CreateActivity(db, models.FactoryPullRequestActivityParams{RunID: root.RunID, Title: "Merge confidence"})
			require.NoError(t, err)
			execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "agent", root.ID, root.ID, configuration)
			require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
			reportCheck := func(id string, score float64, summary string) {
				_, err := factorycomponent.ReportMergeConfidenceCheck(db, order, root.RunID,
					&factoryevents.AutomationRef{AppID: canvas.ID, NodeID: "agent"}, "acme/app", id, score, summary, []string{id}, nil)
				require.NoError(t, err)
			}
			reportCheck("risk", 2, "Limited blast radius.")

			fleet := models.RunnerFleet{ID: uuid.New(), Slug: support.RandomName("poll"), ScopeType: models.RunnerFleetScopeInstallation,
				Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion}
			require.NoError(t, fleet.Create(db))
			now := time.Now()
			agent := models.Runner{ID: uuid.New(), FleetID: fleet.ID, State: models.RunnerStateIdle,
				RunnerVersion: models.DefaultRunnerVersion, CreatedAt: now, UpdatedAt: now}
			require.NoError(t, db.Create(&agent).Error)
			task := models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
				Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: []byte("task"),
				QueuedAt: now, CreatedAt: now, UpdatedAt: now}
			require.NoError(t, db.Create(&task).Error)
			require.NoError(t, task.Reserve(db, agent.ID))
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return task.Start(tx, &agent, "test", now) }))
			taskID := task.ID.String()
			if outcome == "lookup failure" {
				taskID = uuid.NewString()
			}
			require.NoError(t, contexts.NewExecutionRequestContext(db, execution).ScheduleActionCall(runner.HookPoll,
				map[string]any{"task_id": taskID, "organization_id": r.Organization.ID.String()}, time.Minute))
			executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
			nextPoll := func() models.CanvasNodeRequest {
				var pending []models.CanvasNodeRequest
				require.NoError(t, db.Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Find(&pending).Error)
				require.Len(t, pending, 1, "keep one poll scheduled")
				assert.Equal(t, runner.HookPoll, pending[0].Spec.Data().InvokeAction.ActionName)
				assert.Equal(t, taskID, pending[0].Spec.Data().InvokeAction.Parameters["task_id"])
				return pending[0]
			}
			for range 3 {
				request := nextPoll()
				reads := mergeConfidencePollReads(t, executor, request)
				assert.Empty(t, reads, "unfinished tasks and lookup failures must not read report data")
				var completions int64
				require.NoError(t, db.Model(&models.CanvasEvent{}).Where("execution_id = ?", execution.ID).Count(&completions).Error)
				assert.Zero(t, completions)
				replacement := nextPoll()
				assert.NotEqual(t, request.ID, replacement.ID)
				assert.True(t, replacement.RunAt.After(time.Now()))
			}
			if outcome == "lookup failure" {
				return
			}

			// The last check must be stored before task completion makes the report eligible.
			reportCheck("security", 4, "Final security result.")
			exitCode := int32(0)
			channel := "passed"
			if outcome == "failed" {
				exitCode = 1
				channel = "failed"
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				completed, err := task.Complete(tx, &agent, "completion", datatypes.JSON(`{}`), exitCode, "", false, time.Now())
				require.True(t, completed)
				return err
			}))
			reads := mergeConfidencePollReads(t, executor, nextPoll())
			assert.Contains(t, reads, "factory_work_order_events", "terminal polls must read the saved check history")
			var events []models.CanvasEvent
			require.NoError(t, db.Where("execution_id = ?", execution.ID).Find(&events).Error)
			require.Len(t, events, 1)
			assert.Equal(t, channel, events[0].Channel)
			payload := events[0].Data.Data().(map[string]any)["data"].(map[string]any)
			assert.Equal(t, outcome, payload["status"])
			encoded, err := json.Marshal(payload["mergeConfidence"])
			require.NoError(t, err)
			var report mergeconfidence.ReportData
			require.NoError(t, json.Unmarshal(encoded, &report))
			assert.Equal(t, pr.ID.String(), report.PullRequestID)
			assert.Equal(t, "reviewed-head", report.ReviewedCommit)
			assert.Equal(t, outcome == "succeeded", report.Complete)
			assert.Equal(t, []mergeconfidence.ReportCheck{
				{ID: "risk", Label: "Blast radius", Score: 2, MaxScore: 5, Summary: "Limited blast radius.", Level: factoryevents.CheckLevelPositive},
				{ID: "security", Label: "Security", Score: 4, MaxScore: 5, Summary: "Final security result.", Level: factoryevents.CheckLevelPositive},
			}, report.Checks)
			var pending int64
			require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("execution_id = ? AND state = ?", execution.ID, models.NodeExecutionRequestStatePending).Count(&pending).Error)
			assert.Zero(t, pending)
			var run models.CanvasRun
			require.NoError(t, db.First(&run, "id = ?", root.RunID).Error)
			assert.Equal(t, outcome == "failed", len(run.Errors) > 0)
		})
	}
}

func mergeConfidencePollReads(t *testing.T, executor *NodeRequestExecutor, request models.CanvasNodeRequest) []string {
	t.Helper()
	var reads []string
	queries := database.Conn().Callback().Query()
	name := "test:merge-confidence-poll:" + uuid.NewString()
	require.NoError(t, queries.Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if slices.Contains([]string{"workflow_versions", "factory_pull_request_runs", "factory_pull_requests",
			"factory_work_orders", "factory_work_order_checks", "factory_work_order_events"}, tx.Statement.Table) {
			reads = append(reads, tx.Statement.Table)
		}
	}))
	defer func() { require.NoError(t, queries.Remove(name)) }()
	require.NoError(t, executor.RunNow(request))
	return reads
}
