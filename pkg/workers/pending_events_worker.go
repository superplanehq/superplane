package workers

import (
	"encoding/json"
	"errors"
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
	// Calculate field set for event, and check if field set record already exists.
	// If it doesn't, create it.
	//
	fields, hash, err := connectionGroup.CalculateFieldSet(event)
	if err != nil {
		return err
	}

	fieldSet, err := models.FindConnectionGroupFieldSetByHash(tx, connectionGroup.ID, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fieldSet, err = models.CreateConnectionGroupFieldSet(tx, connectionGroup.ID, fields, hash)
			if err != nil {
				return err
			}
		}
	}

	//
	// Create new connection group field set event.
	//
	_, err = models.CreateConnectionGroupFieldSetEvent(tx, fieldSet.ID, event)
	if err != nil {
		return err
	}

	//
	// Check if new event should be emitted, and if so, emit it.
	//
	return w.emitConnectionGroupEvent(tx, connectionGroup, fieldSet)
}

func (w *PendingEventsWorker) emitConnectionGroupEvent(tx *gorm.DB, connectionGroup *models.ConnectionGroup, fieldSet *models.ConnectionGroupFieldSet) error {
	connections, err := models.ListConnectionsInTransaction(tx, connectionGroup.ID, models.ConnectionTargetTypeConnectionGroup)
	if err != nil {
		return fmt.Errorf("error listing connections: %v", err)
	}

	connectionsWithFieldSet, err := models.FindConnectionsWithFieldSetHash(tx, connectionGroup.ID, fieldSet.FieldSetHash)
	if err != nil {
		return fmt.Errorf("error finding connections with field set hash: %v", err)
	}

	//
	// If one of the connections still hasn't emitted an event with this field,
	// we do not emit any event for the connection group.
	//
	for _, conn := range connections {
		if !slices.Contains(connectionsWithFieldSet, conn.SourceID.String()) {
			log.Infof("Event from %s with %v not received for connection group %s", conn.SourceName, fieldSet.FieldSet.Data(), connectionGroup.Name)
			return nil
		}
	}

	log.Infof("All events received for %v - emitting event for connection group %s", fieldSet.FieldSet.Data(), connectionGroup.Name)

	//
	// If we get here, we know that we have received events
	// with all the required fields from all the connections in the group,
	// so we emit an event for it.
	//
	eventData, err := w.buildConnectionGroupEvent(tx, fieldSet)
	if err != nil {
		return fmt.Errorf("error building connection group event: %v", err)
	}

	_, err = models.CreateEventInTransaction(
		tx,
		connectionGroup.ID,
		connectionGroup.Name,
		models.SourceTypeConnectionGroup,
		eventData,
		[]byte(`{}`),
	)

	if err != nil {
		return err
	}

	return fieldSet.UpdateState(tx, models.ConnectionGroupFieldSetStateProcessed)
}

func (w *PendingEventsWorker) buildConnectionGroupEvent(tx *gorm.DB, fieldSet *models.ConnectionGroupFieldSet) ([]byte, error) {
	event := map[string]any{}

	//
	// Include fields from field set.
	//
	for k, v := range fieldSet.FieldSet.Data() {
		event[k] = v
	}

	//
	// Include events from connections.
	//
	events, err := fieldSet.FindEventsWithData(tx)
	if err != nil {
		return nil, err
	}

	for _, e := range events {
		var obj map[string]any
		err := json.Unmarshal(e.Raw, &obj)
		if err != nil {
			return nil, err
		}

		event[e.SourceName] = obj
	}

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
