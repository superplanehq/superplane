package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__RunInitializer__PublishesRunStateWhenInitializationFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "not-a-trigger",
				Type:   models.NodeTypeComponent,
			},
		},
		nil,
	)

	run := createPendingRun(t, canvas.ID, "not-a-trigger", []core.RunCallback{
		{
			When: core.RunCallbackWhenPending,
			On:   core.RunCallbackOnEntry,
			Hook: "onMessage",
		},
	})

	initializer := NewRunInitializer(amqpURL, r.Registry)
	require.NoError(t, initializer.initializeRun(canvas.ID, run.ID, runInitializerTriggerPending))

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultFailed, updatedRun.Result)
	assert.True(t, runConsumer.HasReceivedMessage())
}

func Test__RunInitializer__PublishesRunStateWhenInitializationSucceeds(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	runConsumer := testconsumer.New(amqpURL, messages.CanvasRunRoutingKey)
	runConsumer.Start()
	defer runConsumer.Stop()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "onRun",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "onRun"},
				}),
			},
		},
		nil,
	)

	run := createPendingRun(t, canvas.ID, "onRun", []core.RunCallback{
		{
			When: core.RunCallbackWhenPending,
			On:   core.RunCallbackOnEntry,
			Hook: "onMessage",
		},
	})
	require.NoError(t, database.Conn().Model(run).Update("input", models.NewJSONValue(map[string]any{
		"app": map[string]any{
			"id":   canvas.ID.String(),
			"name": "Source App",
		},
		"parameters": map[string]any{
			"parameter": "hello",
		},
	})).Error)

	initializer := NewRunInitializer(amqpURL, r.Registry)
	require.NoError(t, initializer.initializeRun(canvas.ID, run.ID, runInitializerTriggerPending))

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateStarted, updatedRun.State)
	assert.True(t, runConsumer.HasReceivedMessage())
}

func Test__RunInitializer__FinishesFactoryWorkOrderExecutionWhenInitializationFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "not-a-trigger",
				Type:   models.NodeTypeComponent,
			},
		},
		nil,
	)

	run := createPendingRun(t, canvas.ID, "not-a-trigger", []core.RunCallback{
		{
			When: core.RunCallbackWhenPending,
			On:   core.RunCallbackOnEntry,
			Hook: "onMessage",
		},
	})

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "step-one",
		RunID:          run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)

	amqpURL, _ := config.RabbitMQURL()
	initializer := NewRunInitializer(amqpURL, r.Registry)
	require.NoError(t, initializer.initializeRun(canvas.ID, run.ID, runInitializerTriggerPending))

	updatedRun, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateFinished, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultFailed, updatedRun.Result)

	updatedExecution, err := models.FindWorkOrderExecutionByRunID(database.Conn(), run.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, updatedExecution.Status)
	assert.Equal(t, models.CanvasRunResultFailed, updatedExecution.Result)

	reloadedDispatch, err := models.FindWorkOrderLineDispatch(database.Conn(), dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloadedDispatch.State)
	assert.Equal(t, models.CanvasRunResultFailed, reloadedDispatch.Result)

	_, err = order.FindActiveLineDispatch(database.Conn())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func createPendingRun(t *testing.T, workflowID uuid.UUID, nodeID string, callbacks []core.RunCallback) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		NodeID:     nodeID,
		VersionID:  liveVersion.ID,
		State:      models.CanvasRunStatePending,
		Callbacks:  datatypes.NewJSONSlice(callbacks),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}
