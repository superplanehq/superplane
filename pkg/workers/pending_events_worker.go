package workers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
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

	err = w.enqueueEvent(event, stages)
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

func (w *PendingEventsWorker) enqueueEvent(event *models.Event, stages []models.Stage) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, stage := range stages {
			inputs, err := w.buildInputs(tx, event, stage)
			if err != nil {
				return err
			}

			stageEvent, err := models.CreateStageEventInTransaction(tx, stage.ID, event, models.StageEventStatePending, "", inputs)
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

func (w *PendingEventsWorker) buildInputs(tx *gorm.DB, event *models.Event, stage models.Stage) (map[string]any, error) {
	inputBuilder := inputs.NewBuilder(stage)
	inputs, err := inputBuilder.Build(tx, event)
	if err != nil {
		return nil, err
	}

	return inputs, nil
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
