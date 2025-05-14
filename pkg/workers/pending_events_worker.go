package workers

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type PendingEventsWorker struct{}

func (w *PendingEventsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing pending events: %v", err)
		}

		time.Sleep(time.Second)
	}
}

func (w *PendingEventsWorker) Tick() error {
	events, err := models.ListPendingEvents()
	if err != nil {
		log.Errorf("Error listing pending events: %v", err)
		return err
	}

	for _, event := range events {
		e := event
		logger := logging.ForEvent(&event)
		err := w.ProcessEvent(logger, &e)
		if err != nil {
			logger.Errorf("Error processing pending event: %v", err)
		}
	}

	return nil
}

func (w *PendingEventsWorker) ProcessEvent(logger *log.Entry, event *models.Event) error {
	logger.Info("Processing")

	connections, err := models.ListConnectionsForSource(
		event.SourceID,
		event.SourceType,
	)

	if err != nil {
		return fmt.Errorf("error listing connections: %v", err)
	}

	//
	// If the source is not connected to any stage, we discard the event.
	//
	if len(connections) == 0 {
		logger.Info("Unconnected source - discarding")
		err := event.Discard()
		if err != nil {
			return fmt.Errorf("error discarding event: %v", err)
		}

		return nil
	}

	//
	// Otherwise, we find all the stages, apply their filters on this event.
	//
	stageIDs := w.stageIDsFromConnections(connections)
	stages, err := models.ListStagesByIDs(stageIDs)
	if err != nil {
		return fmt.Errorf("error listing stages: %v", err)
	}

	logger.Infof("Connected stages: %v", stageIDs)

	stages, err = w.filterStages(logger, event, stages, connections)
	if err != nil {
		return fmt.Errorf("error applying filters: %v", err)
	}

	//
	// If after applying the filters,
	// we realize this event shouldn't go to any stage,
	// we mark it as processed, and return.
	//
	if len(stages) == 0 {
		logger.Info("No connections after filtering")
		err := event.MarkAsProcessed()
		if err != nil {
			return fmt.Errorf("error discarding event: %v", err)
		}

		return nil
	}

	err = w.enqueueEvent(logger, event, stages)
	if err != nil {
		return err
	}

	log.Infof("Stages after filtering: %v", w.idsFromStages(stages))
	return nil
}

func findConnectionForStage(stageID string, connections []models.StageConnection) (models.StageConnection, error) {
	for _, connection := range connections {
		if connection.StageID.String() == stageID {
			return connection, nil
		}
	}

	return models.StageConnection{}, fmt.Errorf("connection not found for stage ID: %s", stageID)
}

func (w *PendingEventsWorker) filterStages(logger *log.Entry, event *models.Event, stages []models.Stage, connections []models.StageConnection) ([]models.Stage, error) {
	filtered := []models.Stage{}

	for _, stage := range stages {
		connection, err := findConnectionForStage(stage.ID.String(), connections)
		if err != nil {
			return nil, fmt.Errorf("error finding connection for stage: %v", err)
		}

		//
		// If the filter evaluation fails, we only log the error and skip this stage.
		//
		accept, err := connection.Accept(event)
		if err != nil {
			logger.Errorf("Error applying filter on stage %s: %v", stage.ID, err)
			continue
		}

		if !accept {
			logger.Infof("Not sending to stage %s - filters did not pass", stage.ID)
			continue
		}

		logger.Infof("Sending to stage %s", stage.ID)
		filtered = append(filtered, stage)
	}

	return filtered, nil
}

func (w *PendingEventsWorker) enqueueEvent(logger *log.Entry, event *models.Event, stages []models.Stage) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, stage := range stages {

			//
			// Start by computing our map of tags
			//
			tags, err := w.evaluateTags(logger, event, stage.Use.Data())
			if err != nil {
				return err
			}

			logger.Infof("Evaluated tags: %v", tags)
			stageEvent, err := w.createStageEvent(tx, stage, event, tags)
			if err != nil {
				return err
			}

			err = messages.NewStageEventCreatedMessage(stage.CanvasID.String(), stageEvent).Publish()
			if err != nil {
				logging.ForStage(&stage).Errorf("failed to publish stage event created message: %v", err)
			}
		}

		if err := event.MarkAsProcessedInTransaction(tx); err != nil {
			return fmt.Errorf("error enqueueing event %s: %v", event.ID, err)
		}

		return nil
	})
}

func (w *PendingEventsWorker) createStageEvent(tx *gorm.DB, stage models.Stage, event *models.Event, tags map[string]string) (*models.StageEvent, error) {
	tagUsageDefinition := stage.Use.Data()
	if len(tagUsageDefinition.From) == 1 {
		return w.createStageEventAndTags(tx, stage, event, tags, models.StageEventStatePending, "")
	}

	return w.handleMultipleConnectionTagSource(tx, stage, event, tags)
}

func (w *PendingEventsWorker) createStageEventAndTags(tx *gorm.DB,
	stage models.Stage,
	event *models.Event,
	tags map[string]string,
	state, stateReason string,
) (*models.StageEvent, error) {

	//
	// If we are just concerned with the tags coming from one connection,
	// we can create the stage event in pending state directly.
	//
	stageEvent, err := models.CreateStageEventInTransaction(tx, stage.ID, event, state, stateReason)
	if err != nil {
		return nil, fmt.Errorf("error creating pending stage event: %v", err)
	}

	// TODO: doing it in bulk would be better
	for k, v := range tags {
		err = models.CreateStageEventTagInTransaction(tx, k, v, stageEvent.ID)
		if err != nil {
			return nil, fmt.Errorf("error creating tag value: %v", err)
		}
	}

	return stageEvent, nil
}

func (w *PendingEventsWorker) handleMultipleConnectionTagSource(tx *gorm.DB, stage models.Stage, event *models.Event, tags map[string]string) (*models.StageEvent, error) {
	//
	// List all stage events in waiting(connection) for this
	//
	existingStageEvents, err := stage.ListEventsInTransaction(tx,
		[]string{models.StageEventStateWaiting},
		[]string{models.StageEventStateReasonConnection},
	)

	if err != nil {
		return nil, err
	}

	//
	// If events for (all sources - currentSource) are not there,
	// we create a new stage event in waiting(connection) state for this one.
	//
	from := stage.Use.Data().From
	connections := slices.DeleteFunc(from, func(c string) bool { return c == event.SourceName })
	if !allConnectionsReceived(existingStageEvents, connections) {
		return w.createStageEventAndTags(tx,
			stage,
			event,
			tags,
			models.StageEventStateWaiting,
			models.StageEventStateReasonConnection,
		)
	}

	//
	// If events for (all sources - currentSource) are there,
	// we create a new pending stage event for this one, and
	// move all the previous ones to processed(cancelled).
	//
	stageEvent, err := w.createStageEventAndTags(tx, stage, event, tags, models.StageEventStatePending, "")
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, e := range existingStageEvents {
		ids = append(ids, e.ID.String())
	}

	err = models.UpdateStageEventsInTransaction(tx,
		ids,
		models.StageEventStateProcessed,
		models.StageEventStateReasonCancelled,
	)

	if err != nil {
		return nil, err
	}

	return stageEvent, nil
}

func allConnectionsReceived(events []models.StageEvent, connections []string) bool {
	for _, c := range connections {
		contains := slices.ContainsFunc(events, func(e models.StageEvent) bool {
			return e.SourceName == c
		})

		if !contains {
			return false
		}
	}

	return true
}

func (w *PendingEventsWorker) evaluateTags(logger *log.Entry, event *models.Event, tagUsage models.StageTagUsageDefinition) (map[string]string, error) {

	//
	// If we don't use any tags from this source,
	// no need to do anything regarding tags here.
	//
	if !slices.Contains(tagUsage.From, event.SourceName) {
		logger.Infof("Source %s is not in tag usage definition (%v)", event.SourceName, tagUsage.From)
		return map[string]string{}, nil
	}

	logger.Infof("Processing tags %v...", tagUsage.Tags)

	tagMap := map[string]string{}
	for _, tagDefinition := range tagUsage.Tags {
		value, err := event.EvaluateStringExpression(tagDefinition.ValueFrom)
		if err != nil {
			return nil, fmt.Errorf("error finding tag value for tag %s: %v", tagDefinition.Name, err)
		}

		tagMap[tagDefinition.Name] = value
	}

	return tagMap, nil
}

func (w *PendingEventsWorker) stageIDsFromConnections(connections []models.StageConnection) []uuid.UUID {
	IDs := []uuid.UUID{}
	for _, c := range connections {
		IDs = append(IDs, c.StageID)
	}

	return IDs
}

func (w *PendingEventsWorker) idsFromStages(stages []models.Stage) []uuid.UUID {
	IDs := []uuid.UUID{}
	for _, s := range stages {
		IDs = append(IDs, s.ID)
	}

	return IDs
}
