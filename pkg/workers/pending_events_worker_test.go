package workers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const EventCreatedRoutingKey = "stage-event-created"

func Test__PendingEventsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})

	defer r.Close()
	w := NewPendingEventsWorker(r.Encryptor, r.Registry)

	eventData := []byte(`{"ref":"v1"}`)
	eventHeaders := []byte(`{"ref":"v1"}`)

	executorType, executorSpec, integrationResource := support.Executor(t, r)

	t.Run("source is not connected to any stage -> event is discarded", func(t *testing.T) {
		event, err := models.CreateEvent(r.Source.ID, r.Source.Name, models.SourceTypeEventSource, eventData, eventHeaders)
		require.NoError(t, err)

		err = w.Tick()
		require.NoError(t, err)

		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateDiscarded, event.State)
	})

	t.Run("source is connected to many stages -> event is added to each stage queue", func(t *testing.T) {

		//
		// Create two stages, connecting event source to them.
		//
		inputs := []models.InputDefinition{{Name: "VERSION"}}
		inputMappings := []models.InputMapping{
			{
				Values: []models.ValueDefinition{
					{
						Name: "VERSION",
						ValueFrom: &models.ValueDefinitionFrom{
							EventData: &models.ValueDefinitionFromEventData{
								Connection: r.Source.Name,
								Expression: "ref",
							},
						},
					},
				},
			},
		}

		stage1, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-1").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithInputs(inputs).
			WithInputMappings(inputMappings).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		stage2, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-2").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithInputs(inputs).
			WithInputMappings(inputMappings).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, EventCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		//
		// Create an event for the source, and trigger the worker.
		//
		event, err := models.CreateEvent(r.Source.ID, r.Source.Name, models.SourceTypeEventSource, eventData, eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Event is moved to processed state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)

		//
		// Two pending stage events are created: one for each stage.
		//
		stage1Events, err := stage1.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, stage1Events, 1)
		assert.Equal(t, r.Source.ID, stage1Events[0].SourceID)
		assert.Equal(t, map[string]any{"VERSION": "v1"}, stage1Events[0].Inputs.Data())

		stage2Events, err := stage2.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, stage2Events, 1)
		assert.Equal(t, r.Source.ID, stage2Events[0].SourceID)
		assert.True(t, testconsumer.HasReceivedMessage())
		assert.Equal(t, map[string]any{"VERSION": "v1"}, stage1Events[0].Inputs.Data())
	})

	t.Run("sources are connected to connection group", func(t *testing.T) {
		source2, err := r.Canvas.CreateEventSource("source-2", []byte(`key`), models.EventSourceScopeExternal, nil)
		require.NoError(t, err)

		//
		// Create connection group connected to both sources
		//
		connectionGroup, err := r.Canvas.CreateConnectionGroup(
			"connection-group-1",
			r.User.String(),
			[]models.Connection{
				{SourceID: r.Source.ID, SourceName: r.Source.Name, SourceType: models.SourceTypeEventSource},
				{SourceID: source2.ID, SourceName: source2.Name, SourceType: models.SourceTypeEventSource},
			},
			models.ConnectionGroupSpec{
				GroupBy: &models.ConnectionGroupBySpec{
					Fields: []models.ConnectionGroupByField{
						{Name: "VERSION", Expression: "ref"},
					},
				},
			},
		)

		require.NoError(t, err)

		//
		// Create an event for the first source, and trigger the worker.
		//
		event, err := models.CreateEvent(r.Source.ID, r.Source.Name, models.SourceTypeEventSource, eventData, eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Field set is created, but remains in pending state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)
		fieldSets, err := connectionGroup.ListFieldSets()
		require.NoError(t, err)
		require.Len(t, fieldSets, 1)
		assert.Equal(t, models.ConnectionGroupFieldSetStatePending, fieldSets[0].State)

		//
		// Create an event for the second source, and trigger the worker.
		//
		event, err = models.CreateEvent(source2.ID, source2.Name, models.SourceTypeEventSource, eventData, eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Field set is moved to processed(ok) state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)
		fieldSets, err = connectionGroup.ListFieldSets()
		require.NoError(t, err)
		require.Len(t, fieldSets, 1)
		assert.Equal(t, models.ConnectionGroupFieldSetStateProcessed, fieldSets[0].State)
		assert.Equal(t, models.ConnectionGroupFieldSetStateReasonOK, fieldSets[0].StateReason)
	})

	t.Run("stage completion event is processed", func(t *testing.T) {
		//
		// Create two stages.
		// First stage is connected to event source.
		// Second stage is connected fo first stage.
		//
		firstStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-3").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithInputs([]models.InputDefinition{{Name: "VERSION"}}).
			WithInputMappings([]models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{
							Name: "VERSION",
							ValueFrom: &models.ValueDefinitionFrom{
								EventData: &models.ValueDefinitionFromEventData{
									Connection: r.Source.Name,
									Expression: "ref",
								},
							},
						},
					},
				},
			}).
			WithOutputs([]models.OutputDefinition{
				{
					Name:     "VERSION",
					Required: true,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		secondStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-4").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   firstStage.ID,
					SourceType: models.SourceTypeStage,
				},
			}).
			WithInputs([]models.InputDefinition{{Name: "VERSION"}}).
			WithInputMappings([]models.InputMapping{
				{
					Values: []models.ValueDefinition{
						{
							Name: "VERSION",
							ValueFrom: &models.ValueDefinitionFrom{
								EventData: &models.ValueDefinitionFromEventData{
									Connection: firstStage.Name,
									Expression: "outputs.VERSION",
								},
							},
						},
					},
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		//
		// Simulating a stage completion event coming in for the first stage.
		//
		event, err := models.CreateEvent(firstStage.ID, firstStage.Name, models.SourceTypeStage, []byte(`{"outputs":{"VERSION":"v1"}}`), eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Event is moved to processed state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)

		//
		// No events for the first stage, and one pending event for the second stage.
		//
		events, err := firstStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 0)
		events, err = secondStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, firstStage.ID, events[0].SourceID)
		assert.Equal(t, models.StageEventStatePending, events[0].State)
	})

	t.Run("event is filtered", func(t *testing.T) {
		//
		// Create two stages, connecting event source to them.
		// First stage has a filter that should pass our event,
		// but the second stage has a filter that should not pass.
		//
		firstStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-5").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:       r.Source.ID,
					SourceType:     models.SourceTypeEventSource,
					FilterOperator: models.FilterOperatorAnd,
					Filters: []models.Filter{
						{
							Type: models.FilterTypeData,
							Data: &models.DataFilter{
								Expression: "ref == 'v1'",
							},
						},
					},
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		secondStage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-6").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:       r.Source.ID,
					SourceType:     models.SourceTypeEventSource,
					FilterOperator: models.FilterOperatorAnd,
					Filters: []models.Filter{
						{
							Type: models.FilterTypeData,
							Data: &models.DataFilter{
								Expression: "ref == 'v2'",
							},
						},
					},
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)

		//
		// Create an event for the source, and trigger the worker.
		//
		event, err := models.CreateEvent(r.Source.ID, r.Source.Name, models.SourceTypeEventSource, eventData, eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Event is moved to processed state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)

		//
		// A pending stage event should be created only for the first stage
		//
		events, err := firstStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, r.Source.ID, events[0].SourceID)

		events, err = secondStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 0)
	})

	t.Run("execution resource is updated", func(t *testing.T) {
		//
		// Create pending execution resource
		//
		workflowID := uuid.New().String()
		stage, err := builders.NewStageBuilder(r.Registry).
			WithEncryptor(r.Encryptor).
			InCanvas(r.Canvas).
			WithName("stage-7").
			WithRequester(r.User).
			WithConnections([]models.Connection{
				{
					SourceID:   r.Source.ID,
					SourceType: models.SourceTypeEventSource,
				},
			}).
			WithExecutorType(executorType).
			WithExecutorSpec(executorSpec).
			ForResource(integrationResource).
			ForIntegration(r.Integration).
			Create()

		require.NoError(t, err)
		execution := support.CreateExecution(t, r.Source, stage)
		resource, err := models.FindResource(r.Integration.ID, integrationResource.Type(), integrationResource.Name())
		require.NoError(t, err)
		_, err = execution.AddResource(workflowID, semaphore.ResourceTypeWorkflow, resource.ID)
		require.NoError(t, err)

		//
		// Create a Semaphore hook event for the source created for the execution,
		// and trigger the worker.
		//
		hook := semaphore.Hook{
			Workflow: semaphore.HookWorkflow{
				ID: workflowID,
			},
			Pipeline: semaphore.HookPipeline{
				ID:     uuid.New().String(),
				State:  semaphore.PipelineStateDone,
				Result: semaphore.PipelineResultPassed,
			},
		}

		eventData, err := json.Marshal(hook)
		require.NoError(t, err)
		source, err := resource.FindEventSource()
		require.NoError(t, err)
		event, err := models.CreateEvent(source.ID, source.Name, models.SourceTypeEventSource, eventData, []byte(`{}`))
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Event is discarded, since the event source used by the executor cannot be used as a connection.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateDiscarded, event.State)

		//
		// The execution resource has its state updated.
		//
		resources, err := execution.Resources()
		require.NoError(t, err)
		require.Len(t, resources, 1)
		executionResource := resources[0]
		assert.Equal(t, models.ExecutionFinished, executionResource.State)
		assert.Equal(t, models.ResultPassed, executionResource.Result)
	})
}
