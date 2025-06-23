package workers

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

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

	for _, connection := range connections {
		accept, err := connection.Accept(event)
		if err != nil {
			logger.Errorf("Error applying filter: %v", err)
			continue
		}

		if !accept {
			continue
		}

	}

	connections = w.filterConnections(logger, event, connections)
	if len(connections) == 0 {
		logger.Info("No connections after filtering")
		err := event.MarkAsProcessed()
		if err != nil {
			return fmt.Errorf("error discarding event: %v", err)
		}

		return nil
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, connection := range connections {
			err = w.handleConnectionEvent(tx, event, connection)
			if err != nil {
				return err
			}
		}

		return event.MarkAsProcessedInTransaction(tx)
	})
}

func (w *PendingEventsWorker) filterConnections(logger *log.Entry, event *models.Event, connections []models.Connection) []models.Connection {
	filtered := []models.Connection{}

	for _, connection := range connections {
		accept, err := connection.Accept(event)
		if err != nil {
			logger.Errorf("Error applying filter: %v", err)
			continue
		}

		if !accept {
			continue
		}

		filtered = append(filtered, connection)
	}

	return filtered
}

func (w *PendingEventsWorker) handleConnectionEvent(tx *gorm.DB, event *models.Event, connection models.Connection) error {
	switch connection.TargetType {
	case models.ConnectionTargetTypeStage:
		return w.handleStageEvent(tx, event, connection)

	case models.ConnectionTargetTypeConnectionGroup:
		return w.handleConnectionGroupEvent(tx, event, connection)

	default:
		return fmt.Errorf("invalid target type: %s", connection.TargetType)
	}
}

func (w *PendingEventsWorker) handleStageEvent(tx *gorm.DB, event *models.Event, connection models.Connection) error {
	stage, err := models.FindStageByIDInTransaction(tx, connection.TargetID.String())
	if err != nil {
		return err
	}

	inputs, err := w.buildInputs(tx, event, *stage)
	if err != nil {
		return err
	}

	stageEvent, err := models.CreateStageEventInTransaction(tx, stage.ID, event, models.StageEventStatePending, "", inputs)
	if err != nil {
		return err
	}

	err = messages.NewStageEventCreatedMessage(stage.CanvasID.String(), stageEvent).Publish()
	if err != nil {
		logging.ForStage(stage).Errorf("failed to publish stage event created message: %v", err)
	}

	return nil
}

func (w *PendingEventsWorker) handleConnectionGroupEvent(tx *gorm.DB, event *models.Event, connection models.Connection) error {
	connectionGroup, err := models.FindConnectionGroupByID(tx, connection.TargetID)
	if err != nil {
		return err
	}

	//
	// Create new records for connection group event and keys.
	//
	_, err = models.CreateConnectionGroupEvent(tx, connectionGroup.ID, event)
	if err != nil {
		return fmt.Errorf("error creating connection group event: %v", err)
	}

	keys := []models.ConnectionGroupKey{}
	for _, keyDef := range connectionGroup.Spec.Data().Keys {
		value, err := event.EvaluateStringExpression(keyDef.Expression)
		if err != nil {
			return fmt.Errorf("error evaluating expression '%s' for connection group key %s: %v", keyDef.Expression, keyDef.Name, err)
		}

		keys = append(keys, models.ConnectionGroupKey{
			ConnectionGroupID: connectionGroup.ID,
			SourceID:          event.SourceID,
			Name:              keyDef.Name,
			Value:             value,
		})
	}

	err = tx.Create(keys).Error
	if err != nil {
		return err
	}

	//
	// Check if new event should be emitted, and if so, emit it.
	//
	return w.emitConnectionGroupEvent(tx, connectionGroup, keys)
}

func (w *PendingEventsWorker) emitConnectionGroupEvent(tx *gorm.DB, connectionGroup *models.ConnectionGroup, keys []models.ConnectionGroupKey) error {
	connections, err := models.ListConnectionsInTransaction(tx, connectionGroup.ID, models.ConnectionTargetTypeConnectionGroup)
	if err != nil {
		return fmt.Errorf("error listing connections: %v", err)
	}

	for _, key := range keys {
		connectionsWithKey, err := models.FindConnectionsWithGroupKey(tx, connectionGroup.ID, key.Name, key.Value)
		if err != nil {
			return fmt.Errorf("error finding connections for group key: %v", err)
		}

		//
		// If one of the connections still hasn't emitted an event with this key,
		// we do not emit any event for the connection group.
		//
		for _, conn := range connections {
			if !slices.Contains(connectionsWithKey, conn.SourceID.String()) {
				log.Infof("Event from %s with %s=%s not received for connection group %s", conn.SourceName, key.Name, key.Value, connectionGroup.Name)
				return nil
			}
		}
	}

	//
	// If we get here, we know that we have received events
	// with all the required keys from all the connections in the group,
	// so we emit an event for it.
	//
	eventData, err := w.buildConnectionGroupEvent(keys)
	if err != nil {
		return fmt.Errorf("error building connection group event: %v", err)
	}

	_, err = models.CreateEventInTransaction(
		tx,
		connectionGroup.ID,
		connectionGroup.Name,
		models.SourceTypeConnectionGroup,
		eventData,
		[]byte{},
	)

	return err
}

func (w *PendingEventsWorker) buildConnectionGroupEvent(keys []models.ConnectionGroupKey) ([]byte, error) {
	event := map[string]any{}
	for _, key := range keys {
		event[key.Name] = key.Value
	}

	// TODO: include all events that were grouped into this one

	return json.Marshal(event)
}

func (w *PendingEventsWorker) buildInputs(tx *gorm.DB, event *models.Event, stage models.Stage) (map[string]any, error) {
	inputBuilder := inputs.NewBuilder(stage)
	inputs, err := inputBuilder.Build(tx, event)
	if err != nil {
		return nil, err
	}

	return inputs, nil
}
