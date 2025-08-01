package workers

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type PendingEventsWorker struct {
	Encryptor crypto.Encryptor
	Registry  *registry.Registry
}

func NewPendingEventsWorker(encryptor crypto.Encryptor, registry *registry.Registry) *PendingEventsWorker {
	return &PendingEventsWorker{
		Encryptor: encryptor,
		Registry:  registry,
	}
}

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

		//
		// If anything goes wrong while processing the event, we discard it, and log the error.
		//
		if err != nil {
			logger.Errorf("Error processing event: %v", err)
			err = e.UpdateState(models.EventStateDiscarded)
			if err != nil {
				logger.Errorf("Error discarding event: %v", err)
			}
		}
	}

	return nil
}

func (w *PendingEventsWorker) ProcessEvent(logger *log.Entry, event *models.Event) error {
	logger.Info("Processing")

	//
	// For events coming from event sources (not stages or connection groups),
	// we need to do 2 additional things before processing the connections:
	//
	//   1. Update the state of execution resources that may be connected to this event source.
	//   2. Apply the event source filters
	//
	if event.SourceType == models.SourceTypeEventSource {
		source, err := models.FindEventSource(event.SourceID)
		if err != nil {
			return err
		}

		//
		// If something goes wrong while trying to update execution resources,
		// we just log the error, but proceed with the event processing.
		//
		err = w.UpdateExecutionResource(logger, event, source)
		if err != nil {
			logger.Warnf("Failed to update execution resource: %v", err)
		}

		//
		// If the event does not pass the event source filters,
		// or there's an error evaluating the filters, reject it.
		//
		accept, err := source.Accept(event)
		if err != nil {
			logger.Errorf("Error applying filter: %v", err)
			return err
		}

		if !accept {
			logger.Info("Event does not pass filters - discarding")
			return event.UpdateState(models.EventStateDiscarded)
		}
	}

	//
	// Lastly, we process all the connections for the source of this event.
	//
	return w.ProcessConnections(logger, event)
}

func (w *PendingEventsWorker) UpdateExecutionResource(logger *log.Entry, event *models.Event, source *models.EventSource) error {
	//
	// If this is an event from a stage or connection group,
	// there's nothing to do here.
	//
	if event.SourceType != models.SourceTypeEventSource {
		return nil
	}

	//
	// If this event source is not tied to a resource, there's nothing to do here.
	//
	if source.ResourceID == nil {
		return nil
	}

	resource, err := models.FindResourceByID(*source.ResourceID)
	if err != nil {
		return err
	}

	integration, err := models.FindIntegrationByID(resource.IntegrationID)
	if err != nil {
		return err
	}

	eventHandler, err := w.Registry.GetEventHandler(integration.Type)
	if err != nil {
		return err
	}

	statefulResource, err := eventHandler.Status([]byte(event.Raw))
	if err != nil {
		return err
	}

	if !statefulResource.Finished() {
		return nil
	}

	result := models.ResultPassed
	if !statefulResource.Successful() {
		result = models.ResultFailed
	}

	executionResource, err := models.FindExecutionResource(statefulResource.Id(), *source.ResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("No execution resource %s found for source %s - skipping execution update", statefulResource.Id(), source.ID)
			return nil
		}

		return err
	}

	err = executionResource.Finish(result)
	if err != nil {
		return err
	}

	logger.Infof("Execution resource %s finished with result %s", executionResource.ExternalID, executionResource.Result)

	return nil
}

func (w *PendingEventsWorker) ProcessConnections(logger *log.Entry, event *models.Event) error {
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
		return event.UpdateState(models.EventStateDiscarded)
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
		logger.Info("No connections after applying connection filters - discarding")
		return event.UpdateState(models.EventStateDiscarded)
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, connection := range connections {
			err = w.handleEventForConnection(tx, event, connection)

			//
			// If there is an error handling the event for a single connection,
			// (error building inputs or any other unknown errors), we just log the error,
			// but proceed with the rest of the connections, to avoid blocking the other connections.
			//
			if err != nil {
				logger.Errorf("Error handling event for connection %s (%s): %v", connection.TargetID, connection.TargetType, err)
			}
		}

		return event.UpdateStateInTransaction(tx, models.EventStateProcessed)
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

func (w *PendingEventsWorker) handleEventForConnection(tx *gorm.DB, event *models.Event, connection models.Connection) error {
	switch connection.TargetType {
	case models.ConnectionTargetTypeStage:
		return w.handleEventForStage(tx, event, connection)

	case models.ConnectionTargetTypeConnectionGroup:
		return w.handleEventForConnectionGroup(tx, event, connection)

	default:
		return fmt.Errorf("invalid target type: %s", connection.TargetType)
	}
}

func (w *PendingEventsWorker) handleEventForStage(tx *gorm.DB, event *models.Event, connection models.Connection) error {
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

func (w *PendingEventsWorker) handleEventForConnectionGroup(tx *gorm.DB, event *models.Event, connection models.Connection) error {
	connectionGroup, err := models.FindConnectionGroupByIDInTransaction(tx, connection.TargetID)
	if err != nil {
		return err
	}

	//
	// Calculate field set for event, and check if pending record for it exists.
	// If it doesn't, create it, and attach the event to it.
	//
	fields, hash, err := connectionGroup.CalculateFieldSet(event)
	if err != nil {
		return err
	}

	fieldSet, err := connectionGroup.FindPendingFieldSetByHash(tx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fieldSet, err = connectionGroup.CreateFieldSet(tx, fields, hash)
			if err != nil {
				return err
			}
		}
	}

	_, err = fieldSet.AttachEvent(tx, event)
	if err != nil {
		return err
	}

	//
	// Check if the field set has missing connections.
	// If it does, do not do anything yet.
	// If it doesn't, emit an event for this field set.
	//
	missing, err := fieldSet.MissingConnections(tx, connectionGroup)
	if err != nil {
		return err
	}

	if len(missing) > 0 {
		log.Infof("Connection group %s has missing connections for field set %s: %v",
			connectionGroup.Name, fieldSet.String(), sourceNamesFromConnections(missing),
		)

		return nil
	}

	log.Infof("All connections received for group %s and field set %s - %v", connectionGroup.Name, fieldSet.String(), fields)
	return connectionGroup.EmitInTransaction(tx, fieldSet, models.ConnectionGroupFieldSetStateReasonOK, missing)
}

func (w *PendingEventsWorker) buildInputs(tx *gorm.DB, event *models.Event, stage models.Stage) (map[string]any, error) {
	inputBuilder := inputs.NewBuilder(stage)
	inputs, err := inputBuilder.Build(tx, event)
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

func sourceNamesFromConnections(connections []models.Connection) []string {
	sourceNames := []string{}
	for _, connection := range connections {
		sourceNames = append(sourceNames, connection.SourceName)
	}
	return sourceNames
}
