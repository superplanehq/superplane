package workers

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__EventRouter_ProcessRootEvent(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	terminalEventConsumer := testconsumer.New(amqpURL, messages.EventTerminalRoutingKey)
	terminalEventConsumer.Start()
	defer terminalEventConsumer.Stop()

	//
	// Create a simple canvas with just a trigger and a component nodes.
	//
	node1 := "trigger-1"
	node2 := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node1, Type: models.NodeTypeTrigger},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create the root event for the trigger node, and process it.
	//
	event := support.EmitCanvasEventForNode(t, canvas.ID, node1, "default", nil)
	err := router.LockAndProcessEvent(logger, *event, time.Now())
	require.NoError(t, err)

	//
	// Verify event was marked as routed and
	// new queue item was created for the target node.
	//
	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)
	assert.NotEqual(t, uuid.Nil, updatedEvent.RunID)

	run, err := models.FindCanvasRunByRootEventInTransaction(database.Conn(), event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, run.State)

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, event.ID, queueItems[0].EventID)
	assert.Equal(t, run.ID, queueItems[0].RunID)

	assert.True(t, queueConsumer.HasReceivedMessage())
	assert.True(t, runConsumer.HasReceivedMessage())
	assert.False(t, terminalEventConsumer.HasReceivedMessage())
}

func Test__EventRouter_ProcessRootEvent_RecordsPermanentTaskRejection(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()
	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)
	db := database.Conn()

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Needs an owner", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)
	// The task has no owner, so subscription admission cannot succeed.
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     42,
		URL:        "https://github.com/acme/app/pull/42",
		Title:      "Needs an owner",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	trigger := "trigger-1"
	component := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:   trigger,
				Type:     models.NodeTypeTrigger,
				Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1)),
			},
			{NodeID: component, Type: models.NodeTypeComponent},
		},
		[]models.Edge{{SourceID: trigger, TargetID: component, Channel: "default"}},
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)

	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, trigger, "default", nil, map[string]any{
		"data": map[string]any{"pullRequest": map[string]any{"id": pullRequest.ID.String()}},
	})
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))

	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(db, canvas.ID, component, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	run, err := models.FindCanvasRunByRootEventInTransaction(db, event.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{models.ErrSubscriptionOwnerRequired.Error()}, run.ErrorMessages())

	stored, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	notes, err := stored.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Contains(t, notes[0].Body, models.ErrSubscriptionOwnerRequired.Error())

	subscription = false
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	_, err = order.Close(db, models.FactoryWorkOrderResultArchived, &r.User)
	require.NoError(t, err)
	event = support.EmitCanvasEventForNodeWithData(t, canvas.ID, trigger, "default", nil, map[string]any{
		"data": map[string]any{"pullRequest": map[string]any{"id": pullRequest.ID.String()}},
	})
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))
	updatedEvent, err = models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)
	queueItems, err = models.ListNodeQueueItems(db, canvas.ID, component, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)
}

func Test__EventRouter_ProcessRootEvent_FailsRefinementRunOnPermanentSubscriptionRejection(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()
	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)
	db := database.Conn()

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	// The draft task has no owner, so subscription admission cannot succeed.
	order, err := factory.CreateWorkOrder(db, "Refine without an owner", "", &r.User, nil, nil)
	require.NoError(t, err)

	component := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:   models.FactoryAppBacklogTriggerID,
				Type:     models.NodeTypeTrigger,
				Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 3)),
			},
			{NodeID: component, Type: models.NodeTypeComponent},
		},
		[]models.Edge{{SourceID: models.FactoryAppBacklogTriggerID, TargetID: component, Channel: "default"}},
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)

	event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, models.FactoryAppBacklogTriggerID, "default", nil, map[string]any{
		"type": "workOrder.created",
		"data": map[string]any{"taskRefinementEnabled": true, "workOrder": map[string]any{"id": order.ID.String()}},
	})
	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))

	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(db, canvas.ID, component, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	run, err := models.FindCanvasRunByRootEventInTransaction(db, event.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{models.ErrSubscriptionOwnerRequired.Error()}, run.ErrorMessages())
}

func Test__EventRouter_DoesNotRouteEventForSoftDeletedOrganization(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: triggerNode, Type: models.NodeTypeTrigger},
			{NodeID: componentNode, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	event := support.EmitCanvasEventForNode(t, canvas.ID, triggerNode, "default", nil)
	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))

	events, err := models.ListPendingCanvasEvents()
	require.NoError(t, err)
	for _, pending := range events {
		assert.NotEqual(t, event.ID, pending.ID)
	}

	require.NoError(t, router.LockAndProcessEvent(logger, *event, time.Now()))

	updatedEvent, err := models.FindCanvasEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStatePending, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, componentNode, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, queueItems)

	assert.False(t, queueConsumer.HasReceivedMessage())
}

func Test__EventRouter_ProcessExecutionEvent(t *testing.T) {
	amqpURL, _ := config.RabbitMQURL()

	router := NewEventRouter(amqpURL)
	logger := log.NewEntry(log.New())
	r := support.Setup(t)

	queueConsumer := testconsumer.New(amqpURL, messages.CanvasQueueItemCreatedRoutingKey)
	queueConsumer.Start()
	defer queueConsumer.Stop()

	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	trigger1 := "trigger-1"
	node1 := "component-1"
	node2 := "component-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: trigger1, Type: models.NodeTypeTrigger},
			{NodeID: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: trigger1, TargetID: node1, Channel: "default"},
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	//
	// Create root event for trigger node,
	// and create execution with output event for node1.
	//
	triggerEvent := support.EmitCanvasEventForNode(t, canvas.ID, trigger1, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)
	require.NoError(t, triggerEvent.Routed())
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, node1, triggerEvent.ID, triggerEvent.ID)
	execution.RunID = run.ID
	require.NoError(t, database.Conn().Save(execution).Error)
	_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	//
	// Process node1 output event and verify it was routed properly.
	//
	events, err := models.ListCanvasEvents(database.Conn(), canvas.ID, node1, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	outputEvent := events[0]
	err = router.LockAndProcessEvent(logger, outputEvent, time.Now())
	require.NoError(t, err)

	updatedEvent, err := models.FindCanvasEvent(outputEvent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasEventStateRouted, updatedEvent.State)

	queueItems, err := models.ListNodeQueueItems(database.Conn(), canvas.ID, node2, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	assert.Equal(t, node2, queueItems[0].NodeID)
	assert.Equal(t, outputEvent.ID, queueItems[0].EventID)

	assert.True(t, queueConsumer.HasReceivedMessage())
	assert.False(t, runConsumer.HasReceivedMessage())
}

func Test__EventRouter_RechecksHostedCreditBeforeRoutingPlanningAgent(t *testing.T) {
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	db := database.Conn()
	require.NoError(t, db.Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
			"default_hosted_provider": settings.DefaultHostedProvider,
			"default_hosted_model":    settings.DefaultHostedModel,
		}).Error)
		require.NoError(t, db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}).Error)
	})
	require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
		"default_hosted_provider": nil,
		"default_hosted_model":    nil,
	}).Error)

	triggerID := "trigger"
	gateID := "refinement-enabled"
	agentID := "refine-task"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: triggerID, Type: models.NodeTypeTrigger},
		{NodeID: gateID, Type: models.NodeTypeComponent},
		{
			NodeID: agentID,
			Name:   "Refine Task",
			Type:   models.NodeTypeComponent,
			Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}}),
		},
	}, []models.Edge{
		{SourceID: triggerID, TargetID: gateID, Channel: "default"},
		{SourceID: gateID, TargetID: agentID, Channel: "true"},
	})
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, agentID).
		Updates(map[string]any{
			"state":        models.CanvasNodeStateError,
			"state_reason": models.SuperPlaneRunnerNoCreditMessage,
		}).Error)

	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	routePlanningAttempt := func() *models.CanvasRun {
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerID, "default", nil)
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)
		require.NoError(t, rootEvent.Routed())

		execution := support.CreateCanvasNodeExecution(t, canvas.ID, gateID, rootEvent.ID, rootEvent.ID)
		execution.RunID = run.ID
		require.NoError(t, db.Save(execution).Error)
		_, err = execution.Pass(map[string][]any{"true": {map[string]any{}}})
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(db, canvas.ID, gateID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.NoError(t, router.LockAndProcessEvent(logger, events[0], time.Now()))
		updatedRun, err := models.FindCanvasRunInTransaction(db, canvas.ID, run.ID)
		require.NoError(t, err)
		return updatedRun
	}

	blockedRun := routePlanningAttempt()
	require.Equal(t, []string{models.SuperPlaneRunnerNoCreditMessage}, blockedRun.ErrorMessages())
	queueItems, err := models.ListNodeQueueItems(db, canvas.ID, agentID, 10, nil)
	require.NoError(t, err)
	require.Empty(t, queueItems)

	_, err = billingmodels.AddAdminLLMCreditGrant(db, r.Organization.ID, 1_000_000, "restore credit", nil)
	require.NoError(t, err)
	noModelRun := routePlanningAttempt()
	require.Equal(t, []string{models.SuperPlaneRunnerNoModelMessage}, noModelRun.ErrorMessages())
	queueItems, err = models.ListNodeQueueItems(db, canvas.ID, agentID, 10, nil)
	require.NoError(t, err)
	require.Empty(t, queueItems)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
		"default_hosted_provider": models.UsageProviderAnthropic,
		"default_hosted_model":    "claude-sonnet-4-6",
	}).Error)

	readyRun := routePlanningAttempt()
	require.Empty(t, readyRun.ErrorMessages())
	queueItems, err = models.ListNodeQueueItems(db, canvas.ID, agentID, 10, nil)
	require.NoError(t, err)
	require.Len(t, queueItems, 1)
	require.Equal(t, readyRun.ID, queueItems[0].RunID)
	agent, err := models.FindCanvasNode(db, canvas.ID, agentID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeStateReady, agent.State)
	require.Nil(t, agent.StateReason)
}

func Test__EventRouter_RoutesEventWhenBlockedNodeErrorsAreFull(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	triggerID := "trigger"
	sourceID := "source"
	blockedID := "blocked"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: triggerID, Type: models.NodeTypeTrigger},
		{NodeID: sourceID, Type: models.NodeTypeComponent},
		{NodeID: blockedID, Name: "Blocked", Type: models.NodeTypeComponent},
	}, []models.Edge{
		{SourceID: triggerID, TargetID: sourceID, Channel: "default"},
		{SourceID: sourceID, TargetID: blockedID, Channel: "default"},
	})
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, blockedID).
		Updates(map[string]any{"state": models.CanvasNodeStateError, "state_reason": "node is broken"}).Error)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	require.NoError(t, rootEvent.Routed())
	for i := 0; i < models.MaxRunErrorsCount; i++ {
		require.NoError(t, run.AddError(db, fmt.Sprintf("error %d", i), config.MaxPayloadSize()))
	}

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, sourceID, rootEvent.ID, rootEvent.ID)
	execution.RunID = run.ID
	require.NoError(t, db.Save(execution).Error)
	_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
	require.NoError(t, err)

	events, err := models.ListCanvasEvents(db, canvas.ID, sourceID, 1, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	require.NoError(t, NewEventRouter("").LockAndProcessEvent(log.NewEntry(log.New()), events[0], time.Now()))

	var routed models.CanvasEvent
	require.NoError(t, db.Where("id = ?", events[0].ID).First(&routed).Error)
	assert.Equal(t, models.CanvasEventStateRouted, routed.State)
	updatedRun, err := models.FindCanvasRunInTransaction(db, canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Len(t, updatedRun.Errors, models.MaxRunErrorsCount)
}

func Test__EventRouter_RecordsBlockedNodeErrorOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	triggerID := "trigger"
	sourceID := "source"
	blockedID := "blocked"
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: triggerID, Type: models.NodeTypeTrigger},
		{NodeID: sourceID, Type: models.NodeTypeComponent},
		{NodeID: blockedID, Name: "Blocked", Type: models.NodeTypeComponent},
	}, []models.Edge{
		{SourceID: triggerID, TargetID: sourceID, Channel: "default"},
		{SourceID: sourceID, TargetID: blockedID, Channel: "default"},
	})
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, blockedID).
		Updates(map[string]any{"state": models.CanvasNodeStateError, "state_reason": "node is broken"}).Error)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, triggerID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
	require.NoError(t, err)
	require.NoError(t, rootEvent.Routed())

	router := NewEventRouter("")
	logger := log.NewEntry(log.New())
	for i := 0; i < 2; i++ {
		execution := support.CreateCanvasNodeExecution(t, canvas.ID, sourceID, rootEvent.ID, rootEvent.ID)
		execution.RunID = run.ID
		require.NoError(t, db.Save(execution).Error)
		_, err = execution.Pass(map[string][]any{"default": {map[string]any{}}})
		require.NoError(t, err)

		var pending []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ? AND node_id = ? AND state = ?", canvas.ID, sourceID, models.CanvasEventStatePending).Find(&pending).Error)
		require.Len(t, pending, 1)
		require.NoError(t, router.LockAndProcessEvent(logger, pending[0], time.Now()))
	}

	updatedRun, err := models.FindCanvasRunInTransaction(db, canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"node is broken"}, updatedRun.ErrorMessages())
}
