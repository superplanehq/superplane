package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	testconsumer "github.com/superplanehq/superplane/test/test_consumer"
)

const EventCreatedRoutingKey = "stage-event-created"

func Test__PendingEventsWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{Source: true})
	w := PendingEventsWorker{}

	eventData := []byte(`{"ref":"v1"}`)
	eventHeaders := []byte(`{"ref":"v1"}`)

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
		err := r.Canvas.CreateStage("stage-1", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{
			{
				Name: "VERSION",
			},
		}, []models.InputMapping{
			{
				Values: []models.InputValueDefinition{
					{
						Name: "VERSION",
						ValueFrom: &models.InputValueFrom{
							EventData: &models.InputValueFromEventData{
								Connection: r.Source.Name,
								Expression: "ref",
							},
						},
					},
				},
			},
		}, []models.OutputDefinition{})

		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-2", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{
			{
				Name: "VERSION",
			},
		}, []models.InputMapping{
			{
				Values: []models.InputValueDefinition{
					{
						Name: "VERSION",
						ValueFrom: &models.InputValueFrom{
							EventData: &models.InputValueFromEventData{
								Connection: r.Source.Name,
								Expression: "ref",
							},
						},
					},
				},
			},
		}, []models.OutputDefinition{})

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
		stage1, _ := r.Canvas.FindStageByName("stage-1")
		stage2, _ := r.Canvas.FindStageByName("stage-2")

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

	t.Run("stage completion event is processed", func(t *testing.T) {
		//
		// Create two stages.
		// First stage is connected to event source.
		// Second stage is connected fo first stage.
		//
		err := r.Canvas.CreateStage("stage-3", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, []models.InputDefinition{
			{
				Name: "VERSION",
			},
		}, []models.InputMapping{
			{
				Values: []models.InputValueDefinition{
					{
						Name: "VERSION",
						ValueFrom: &models.InputValueFrom{
							EventData: &models.InputValueFromEventData{
								Connection: r.Source.Name,
								Expression: "ref",
							},
						},
					},
				},
			},
		}, []models.OutputDefinition{
			{
				Name:     "VERSION",
				Required: true,
			},
		})

		require.NoError(t, err)
		firstStage, err := r.Canvas.FindStageByName("stage-3")
		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-4", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:   firstStage.ID,
				SourceType: models.SourceTypeStage,
			},
		}, []models.InputDefinition{
			{
				Name: "VERSION",
			},
		}, []models.InputMapping{
			{
				Values: []models.InputValueDefinition{
					{
						Name: "VERSION",
						ValueFrom: &models.InputValueFrom{
							EventData: &models.InputValueFromEventData{
								Connection: firstStage.Name,
								Expression: "outputs.VERSION",
							},
						},
					},
				},
			},
		}, []models.OutputDefinition{})

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
		secondStage, _ := r.Canvas.FindStageByName("stage-4")
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
		err := r.Canvas.CreateStage("stage-5", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:       r.Source.ID,
				SourceType:     models.SourceTypeEventSource,
				FilterOperator: models.FilterOperatorAnd,
				Filters: []models.StageConnectionFilter{
					{
						Type: models.FilterTypeData,
						Data: &models.DataFilter{
							Expression: "ref == 'v1'",
						},
					},
				},
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{})

		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-6", r.User.String(), []models.StageCondition{}, support.ExecutorSpec(), []models.StageConnection{
			{
				SourceID:       r.Source.ID,
				SourceType:     models.SourceTypeEventSource,
				FilterOperator: models.FilterOperatorAnd,
				Filters: []models.StageConnectionFilter{
					{
						Type: models.FilterTypeData,
						Data: &models.DataFilter{
							Expression: "ref == 'v2'",
						},
					},
				},
			},
		}, []models.InputDefinition{}, []models.InputMapping{}, []models.OutputDefinition{})

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

		firstStage, _ := r.Canvas.FindStageByName("stage-5")
		events, err := firstStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, r.Source.ID, events[0].SourceID)

		secondStage, _ := r.Canvas.FindStageByName("stage-6")
		events, err = secondStage.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 0)
	})
}
