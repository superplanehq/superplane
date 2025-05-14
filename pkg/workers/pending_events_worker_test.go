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
		err := r.Canvas.CreateStage("stage-1", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name))

		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-2", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name))

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

		stage2Events, err := stage2.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, stage2Events, 1)
		assert.Equal(t, r.Source.ID, stage2Events[0].SourceID)
		assert.True(t, testconsumer.HasReceivedMessage())

		//
		// Tag is created for each stage event too
		//
		e := stage1Events[0]
		tags, err := models.FindStageEventTags(e.ID)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, []models.StageEventTag{
			{Name: "VERSION", Value: "v1", State: models.TagStateUnknown, StageEventID: e.ID},
		}, tags)

		e = stage2Events[0]
		tags, err = models.FindStageEventTags(e.ID)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, []models.StageEventTag{
			{Name: "VERSION", Value: "v1", State: models.TagStateUnknown, StageEventID: e.ID},
		}, tags)
	})

	t.Run("stage completion event is processed", func(t *testing.T) {
		//
		// Create two stages.
		// First stage is connected to event source.
		// Second stage is connected fo first stage.
		//
		err := r.Canvas.CreateStage("stage-3", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name))

		require.NoError(t, err)
		firstStage, err := r.Canvas.FindStageByName("stage-3")
		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-4", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   firstStage.ID,
				SourceType: models.SourceTypeStage,
			},
		}, models.StageTagUsageDefinition{
			From: []string{firstStage.Name},
			Tags: []models.StageTagDefinition{
				{Name: "VERSION", ValueFrom: "tags.VERSION"},
			},
		})

		require.NoError(t, err)

		//
		// Simulating a stage completion event coming in for the first stage.
		//
		event, err := models.CreateEvent(firstStage.ID, firstStage.Name, models.SourceTypeStage, []byte(`{"tags":{"VERSION":"v1"}}`), eventHeaders)
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
		err := r.Canvas.CreateStage("stage-5", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
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
		}, support.TagUsageDef(r.Source.Name))

		require.NoError(t, err)

		err = r.Canvas.CreateStage("stage-6", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
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
		}, support.TagUsageDef(r.Source.Name))

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

	t.Run("tag usage definition with multiple from", func(t *testing.T) {
		//
		// Create two stages, connecting event source to them.
		// First stage has a filter that should pass our event,
		// but the second stage has a filter that should not pass.
		//
		err := r.Canvas.CreateStage("preprod1", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name))
		require.NoError(t, err)
		preprod1, err := r.Canvas.FindStageByName("preprod1")
		require.NoError(t, err)

		err = r.Canvas.CreateStage("preprod2", r.User.String(), []models.StageCondition{}, support.RunTemplate(), []models.StageConnection{
			{
				SourceID:   r.Source.ID,
				SourceType: models.SourceTypeEventSource,
			},
		}, support.TagUsageDef(r.Source.Name))
		require.NoError(t, err)
		preprod2, err := r.Canvas.FindStageByName("preprod2")
		require.NoError(t, err)

		//
		// Create third stage, connected to the previous two
		//
		err = r.Canvas.CreateStage(
			"prod",
			r.User.String(),
			[]models.StageCondition{},
			support.RunTemplate(),
			[]models.StageConnection{
				{
					SourceID:   preprod1.ID,
					SourceType: models.SourceTypeStage,
				},
				{
					SourceID:   preprod2.ID,
					SourceType: models.SourceTypeStage,
				},
			},
			models.StageTagUsageDefinition{
				From: []string{preprod1.Name, preprod2.Name},
				Tags: []models.StageTagDefinition{
					{Name: "VERSION", ValueFrom: "tags.VERSION"},
				},
			},
		)

		require.NoError(t, err)
		prod, err := r.Canvas.FindStageByName("prod")
		require.NoError(t, err)

		//
		// Simulating a stage completion event coming in for the preprod1 stage.
		//
		event, err := models.CreateEvent(preprod1.ID, preprod1.Name, models.SourceTypeStage, []byte(`{"tags":{"VERSION":"v1"}}`), eventHeaders)
		require.NoError(t, err)
		err = w.Tick()
		require.NoError(t, err)

		//
		// Event is moved to processed state.
		// Stage event is created in waiting(connection) state.
		//
		event, err = models.FindEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, models.EventStateProcessed, event.State)
		events, err := prod.ListPendingEvents()
		require.NoError(t, err)
		require.Empty(t, events)
		events, err = prod.ListEvents([]string{models.StageEventStateWaiting}, []string{models.StageEventStateReasonConnection})
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, preprod1.ID, events[0].SourceID)

		//
		// Simulating a stage completion event coming in for the preprod2 stage.
		//
		event, err = models.CreateEvent(preprod2.ID, preprod2.Name, models.SourceTypeStage, []byte(`{"tags":{"VERSION":"v1"}}`), eventHeaders)
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
		// Verify that new pending stage event is created for the preprod2 event
		//
		events, err = prod.ListPendingEvents()
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, preprod2.ID, events[0].SourceID)

		//
		// Verify that stage event for preprod1 is moved to processed(cancelled).
		//
		events, err = prod.ListEvents([]string{models.StageEventStateWaiting}, []string{models.StageEventStateReasonConnection})
		require.NoError(t, err)
		require.Len(t, events, 0)
		events, err = prod.ListEvents([]string{models.StageEventStateProcessed}, []string{models.StageEventStateReasonCancelled})
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, preprod1.ID, events[0].SourceID)
	})
}
