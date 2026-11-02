package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factorypb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/registry"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__NodeRequestExecutor_CustomScheduleLifecycle(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("scheduled"), "", "")
	require.NoError(t, err)
	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	repository, branch, integrationID := "acme/app", "main", integration.ID.String()
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository, DefaultBranch: &branch, VCSIntegrationID: &integrationID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	llmSettings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.Background()).Model(llmSettings).Updates(map[string]any{
			"default_hosted_provider": llmSettings.DefaultHostedProvider, "default_hosted_model": llmSettings.DefaultHostedModel,
		}).Error)
	})
	require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", llmSettings.ID).Updates(map[string]any{
		"default_hosted_provider": nil, "default_hosted_model": nil,
	}).Error)
	probe := &scheduledRunnerProbe{Action: r.Registry.Actions["runnerClaudeCode"]}
	r.Registry.Actions["runnerClaudeCode"] = probe
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	deps := factoryactions.IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
	settings := &factorypb.FactoryCustomAutomation_Settings{
		StartMode: factorypb.FactoryCustomAutomation_START_MODE_SCHEDULED,
		Schedule:  &factorypb.FactoryCustomAutomation_Schedule{CronExpression: "0 9 * * *"},
		Agent:     &factorypb.FactoryAppTemplateAgent{Component: "runnerClaudeCode", CredentialSource: "integration", CredentialIntegrationName: integration.InstallationName},
		Steps:     []*factorypb.FactoryCustomAutomation_Step{{Name: "Test", Type: factorypb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make test", WorkingDirectory: "repo"}},
	}
	created, err := factoryactions.CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &factorypb.CreateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), Name: "Scheduled tests", Settings: settings})
	require.NoError(t, err)
	canvasID := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
	requestForLiveTrigger := func() models.CanvasNodeRequest {
		canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvasID)
		require.NoError(t, err)
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		require.NoError(t, err)
		for _, node := range version.Nodes {
			if node.Type != models.NodeTypeTrigger {
				continue
			}
			var request models.CanvasNodeRequest
			var pending int64
			require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("workflow_id = ? AND node_id = ? AND state = ?", canvasID, node.ID, models.NodeExecutionRequestStatePending).Count(&pending).Error)
			assert.EqualValues(t, 1, pending, "one active scheduling chain")
			require.NoError(t, db.Where("workflow_id = ? AND node_id = ? AND state = ?", canvasID, node.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
			return request
		}
		t.Fatal("schedule trigger is missing")
		return models.CanvasNodeRequest{}
	}
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	first := requestForLiveTrigger()
	require.NoError(t, executor.RunNow(first))
	var event models.CanvasEvent
	require.NoError(t, db.Where("workflow_id = ?", canvasID).First(&event).Error)
	amqpURL, _ := config.RabbitMQURL()
	require.NoError(t, NewEventRouter(amqpURL).LockAndProcessEvent(executor.logger, event, time.Now()))
	items, err := models.ListNodeQueueItems(db, canvasID, "verify", 10, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, event.RunID, items[0].RunID)
	payload := models.RootEventSourcePayload(event.Data.Data())
	assert.NotContains(t, payload, "pullRequest")
	assert.NotContains(t, payload, "workOrder")
	var linked int64
	require.NoError(t, db.Model(&models.FactoryPullRequestRun{}).Where("run_id = ?", event.RunID).Count(&linked).Error)
	assert.Zero(t, linked)
	node, err := models.FindCanvasNode(db, canvasID, "verify")
	require.NoError(t, err)
	require.NoError(t, NewNodeQueueWorker(r.Registry, amqpURL).LockAndProcessNode(executor.logger, *node, time.Now()))
	var execution models.CanvasNodeExecution
	require.NoError(t, db.Where("workflow_id = ? AND root_event_id = ?", canvasID, event.ID).First(&execution).Error)
	require.NoError(t, newTestNodeExecutor(t, r).LockAndProcessNodeExecution(execution.ID))
	storedExecution, err := models.FindNodeExecution(canvasID, execution.ID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeExecutionResultPassed, storedExecution.Result, storedExecution.ResultMessage)
	require.True(t, probe.executed, "scheduler tick must reach the generated runner action")
	configuration := probe.configuration.(map[string]any)
	for _, raw := range configuration["environment"].([]any) {
		variable := raw.(map[string]any)
		assert.NotEqual(t, "PR_NUMBER", variable["name"])
		assert.NotEqual(t, "PR_HEAD", variable["name"])
	}
	old := requestForLiveTrigger()
	settings.Schedule.CronExpression = "30 0 9 * * *"
	_, err = factoryactions.UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &factorypb.UpdateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: canvasID.String(), Name: "Scheduled tests", Settings: settings})
	require.NoError(t, err)
	current := requestForLiveTrigger()
	assert.NotEqual(t, old.NodeID, current.NodeID)
	require.NoError(t, executor.RunNow(old))
	count, err := models.CountCanvasEvents(db, canvasID, old.NodeID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
	r.Registry.Integrations["github"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: "github.onPullRequest"})}})
	settings.StartMode, settings.Schedule = factorypb.FactoryCustomAutomation_START_MODE_MANUAL, nil
	_, err = factoryactions.UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &factorypb.UpdateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: canvasID.String(), Name: "Scheduled tests", Settings: settings})
	require.NoError(t, err)
	require.NoError(t, executor.RunNow(current))
	count, err = models.CountCanvasEvents(db, canvasID, current.NodeID)
	require.NoError(t, err)
	assert.Zero(t, count)
	settings.StartMode = factorypb.FactoryCustomAutomation_START_MODE_SCHEDULED
	settings.Schedule = &factorypb.FactoryCustomAutomation_Schedule{CronExpression: "0 10 * * *"}
	_, err = factoryactions.UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &factorypb.UpdateFactoryCustomAutomationRequest{FactoryId: factory.ID.String(), AutomationId: canvasID.String(), Name: "Scheduled tests", Settings: settings})
	require.NoError(t, err)
	current = requestForLiveTrigger()
	_, err = factoryactions.DeleteFactoryAutomation(ctx, r.Organization.ID.String(), &factorypb.DeleteFactoryAutomationRequest{FactoryId: factory.ID.String(), AutomationId: canvasID.String()})
	require.NoError(t, err)
	require.NoError(t, executor.RunNow(current))
	count, err = models.CountCanvasEvents(db, canvasID, current.NodeID)
	require.NoError(t, err)
	assert.Zero(t, count)
}

// Keep the runner schema and setup validation. Replace external task dispatch.
type scheduledRunnerProbe struct {
	core.Action
	executed      bool
	configuration any
}

func (p *scheduledRunnerProbe) Execute(ctx core.ExecutionContext) error {
	p.executed = true
	p.configuration = ctx.Configuration
	return ctx.ExecutionState.Pass()
}

func Test__NodeRequestExecutor_RunNow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "schedule", Type: models.NodeTypeTrigger,
		Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
		Configuration: datatypes.NewJSONType(map[string]any{
			"type": "days", "daysInterval": 1, "hour": 12, "minute": 0,
		}),
	}}, nil)

	for _, name := range []string{"future request", "locked request", "completed request", "failed hook"} {
		t.Run(name, func(t *testing.T) {
			request := models.CanvasNodeRequest{
				ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "schedule",
				Type: models.NodeRequestTypeInvokeAction, State: models.NodeExecutionRequestStatePending,
				RunAt: time.Now().Add(time.Hour),
				Spec:  datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "emitEvent"}}),
			}
			if name == "completed request" {
				request.State = models.NodeExecutionRequestStateCompleted
			}
			if name == "failed hook" {
				request.Spec = datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "missing"}})
			}
			require.NoError(t, db.Create(&request).Error)
			if name == "locked request" {
				tx := db.Begin()
				require.NoError(t, tx.Error)
				defer tx.Rollback()
				_, err := models.LockNodeRequestNow(tx, request.ID)
				require.NoError(t, err)
			}
			var before int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&before).Error)
			done := make(chan error, 1)
			go func() { done <- executor.RunNow(request) }()
			select {
			case err := <-done:
				if name == "failed hook" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("RunNow must skip requests held by another executor")
			}
			updated, err := models.FindCanvasNodeRequest(db, request.ID)
			require.NoError(t, err)
			var after int64
			require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&after).Error)
			if name == "future request" {
				assert.Equal(t, models.NodeExecutionRequestStateCompleted, updated.State)
				assert.Equal(t, before+1, after)
				return
			}
			assert.Equal(t, request.State, updated.State)
			assert.WithinDuration(t, request.RunAt, updated.RunAt, time.Microsecond)
			assert.Equal(t, before, after)
		})
	}
}

func Test__NodeRequestExecutor_InvokeTriggerAction_DefersRunTitleResolutionUntilEventEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
					"customName":   "Minute {{ root().data.calendar.minute }}",
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	var event models.CanvasEvent
	err = database.Conn().
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", triggerNode).
		First(&event).Error
	require.NoError(t, err)
	require.NotNil(t, event.CustomName)
	assert.Regexp(t, `^Minute \d{2}$`, *event.CustomName)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_InvokeNodeComponentActionWithoutExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     componentNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "non-existent-action",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for action noop")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_PreventsConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Have two workers call Execute concurrently on the same request.
	// Execute uses a transaction with locking, so only one should actually process.
	//
	results := make(chan error, 2)

	//
	// Create two workers and have them try to process the request concurrently.
	//
	go func() {
		executor1 := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
		results <- executor1.Execute(request)
	}()

	go func() {
		executor2 := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
		results <- executor2.Execute(request)
	}()

	// Collect results - both should succeed (return nil)
	// because Execute returns nil when it can't acquire the lock
	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	//
	// Verify the request was marked as completed.
	//
	var updatedRequest models.CanvasNodeRequest
	err := database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	//
	// Verify that exactly one workflow event was emitted (proving only one executor processed it).
	//
	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Equal(t, int64(1), eventCount, "Expected exactly 1 workflow event, but found %d", eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_UnsupportedRequestType(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request with an unsupported type.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       "unsupported-type",
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}),
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported node execution request type")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_MissingInvokeActionSpec(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request without an InvokeAction spec.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec:       datatypes.NewJSONType(models.NodeExecutionRequestSpec{}), // Missing InvokeAction
		State:      models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spec is not specified")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_NonExistentTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a trigger node that references a non-existent trigger.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "non-existent-trigger"}}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a trigger action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trigger non-existent-trigger not registered")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_NonExistentAction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for invoking a non-existent action.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "non-existent-action",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Process the request and verify it returns an error.
	//
	err := executor.Execute(request)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook non-existent-action not found for trigger schedule")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_CompletesDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	triggerNode := "trigger-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	requests, err := models.ListNodeRequests()
	require.NoError(t, err)

	found := false
	for _, req := range requests {
		if req.ID == request.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Request for deleted node should be returned by ListNodeRequests")

	err = executor.Execute(request)
	require.NoError(t, err)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_CancelsExecutionForDeletedNodeRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentNode := "component-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateStarted,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "noop",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)
	require.NoError(t, terminator.LockAndCancelExecution(updatedExecution))

	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_CompletesRequestForFinishedExecutionWithoutInvokingHook(t *testing.T) {
	hookCalled := false
	componentName := "finished_execution_request_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})

	finishedAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"updated_at": finishedAt,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, hookCalled)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)
}

func Test__NodeRequestExecutor_CompletesRequestForCancellingExecutionWithoutInvokingHook(t *testing.T) {
	hookCalled := false
	componentName := "cancelling_execution_request_" + uuid.New().String()
	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			hookCalled = true
			return nil
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})

	cancellingAt := time.Now().Add(-10 * time.Minute)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateCancelling,
		"updated_at": cancellingAt,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, hookCalled)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, cancellingAt, *updatedExecution.UpdatedAt, time.Second)
}

func Test__NodeRequestExecutor_DoesNotSaveStaleExecutionAfterHookFindsPersistedExecutionFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentName := "stale_execution_after_hook_" + uuid.New().String()
	var executionID uuid.UUID
	finishedAt := time.Now().Add(-10 * time.Minute)
	r.Registry.Actions[componentName] = impl.NewDummyAction(impl.DummyActionOptions{
		Name:  componentName,
		Hooks: []core.Hook{{Name: "poll", Type: core.HookTypeInternal}},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			err := database.Conn().Model(&models.CanvasNodeExecution{}).
				Where("id = ?", executionID).
				Updates(map[string]any{
					"state":      models.CanvasNodeExecutionStateFinished,
					"result":     models.CanvasNodeExecutionResultPassed,
					"updated_at": finishedAt,
				}).Error
			require.NoError(t, err)

			return ctx.ExecutionState.Pass()
		},
	})

	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	executionID = execution.ID

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "poll",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	var updatedExecution models.CanvasNodeExecution
	require.NoError(t, database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, updatedExecution.Result)
	require.NotNil(t, updatedExecution.UpdatedAt)
	assert.WithinDuration(t, finishedAt, *updatedExecution.UpdatedAt, time.Second)

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_CancelsExecutionForDeletedNodeWhenComponentCancelFails(t *testing.T) {
	cancelCalled := false
	componentName := "cancel_failing_" + uuid.New().String()

	registry.RegisterAction(componentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: componentName,
		CancelFunc: func(ctx core.ExecutionContext) error {
			cancelCalled = true
			return errors.New("cancel failed")
		},
	}))

	r := support.Setup(t)
	defer r.Close()
	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)

	componentNode := "component-1"
	canvas, canvasNodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, componentNode, "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID, map[string]any{})
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state": models.CanvasNodeExecutionStateStarted,
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      componentNode,
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "noop",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, database.Conn().Delete(&canvasNodes[0]).Error)

	err := executor.Execute(request)
	require.NoError(t, err)

	assert.False(t, cancelCalled, "NodeRequestExecutor should not invoke component Cancel directly")

	var updatedExecution models.CanvasNodeExecution
	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)

	terminator := NewExecutionTerminator("", r.AuthService, r.Encryptor, r.Registry)
	require.NoError(t, terminator.LockAndCancelExecution(updatedExecution))

	assert.True(t, cancelCalled, "component Cancel should be invoked by ExecutionTerminator")

	err = database.Conn().Where("id = ?", execution.ID).First(&updatedExecution).Error
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)
	assert.Equal(t, models.CanvasNodeExecutionResultCancelled, updatedExecution.Result)

	var updatedRequest models.CanvasNodeRequest
	err = database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error
	require.NoError(t, err)
	assert.Equal(t, models.NodeExecutionRequestStateCompleted, updatedRequest.State)
}

func Test__NodeRequestExecutor_DoesNotProcessDeletedWorkflowRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	//
	// Create a simple canvas with a schedule trigger node.
	//
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]interface{}{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create a node request for the trigger.
	//
	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]interface{}{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	//
	// Soft delete the entire workflow.
	//
	require.NoError(t, database.Conn().Delete(&canvas).Error)

	//
	// Verify that ListNodeRequests does not return the request for the deleted workflow.
	//
	requests, err := models.ListNodeRequests()
	require.NoError(t, err)

	// Check that our request is not in the list
	found := false
	for _, req := range requests {
		if req.ID == request.ID {
			found = true
			break
		}
	}
	assert.False(t, found, "Request for deleted workflow should not be returned by ListNodeRequests")

	assert.False(t, executionConsumer.HasReceivedMessage())
}

func Test__NodeRequestExecutor_DoesNotProcessSoftDeletedOrganizationRequests(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.NewExecutions(amqpURL, messages.ExecutionPendingRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	executor := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService)
	triggerNode := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"type":         "days",
					"daysInterval": 1,
					"hour":         12,
					"minute":       0,
				}),
			},
		},
		[]models.Edge{},
	)

	request := models.CanvasNodeRequest{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     triggerNode,
		Type:       models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "emitEvent",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	requests, err := models.ListNodeRequests()
	require.NoError(t, err)
	for _, pending := range requests {
		assert.NotEqual(t, request.ID, pending.ID)
	}

	require.NoError(t, executor.Execute(request))

	var updatedRequest models.CanvasNodeRequest
	require.NoError(t, database.Conn().Where("id = ?", request.ID).First(&updatedRequest).Error)
	assert.Equal(t, models.NodeExecutionRequestStatePending, updatedRequest.State)

	eventCount, err := models.CountCanvasEvents(database.Conn(), canvas.ID, triggerNode)
	require.NoError(t, err)
	assert.Zero(t, eventCount)
	assert.False(t, executionConsumer.HasReceivedMessage())
}
