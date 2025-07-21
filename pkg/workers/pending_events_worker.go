package workers

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type PendingEventsWorker struct {
	Encryptor crypto.Encryptor
}

func NewPendingEventsWorker(encryptor crypto.Encryptor) *PendingEventsWorker {
	return &PendingEventsWorker{
		Encryptor: encryptor,
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
		if err != nil {
			logger.Errorf("Error processing pending event: %v", err)
		}
	}

	return nil
}

func (w *PendingEventsWorker) ProcessEvent(logger *log.Entry, event *models.Event) error {
	logger.Info("Processing")

	//
	// First, we check if there's an execution resource
	// to update for this event
	//
	err := w.UpdateExecutionResource(logger, event)
	if err != nil {
		return err
	}

	//
	// Lastly, we process all the connections for the source of this event.
	//
	return w.ProcessConnections(logger, event)
}

func (w *PendingEventsWorker) UpdateExecutionResource(logger *log.Entry, event *models.Event) error {
	//
	// If this is an event from a stage or connection group,
	// there's nothing to do here.
	//
	if event.SourceType != models.SourceTypeEventSource {
		return nil
	}

	eventSource, err := models.FindEventSource(event.SourceID)
	if err != nil {
		return err
	}

	//
	// If this event source is not tied to a resource, there's nothing to do here.
	//
	if eventSource.ResourceID == nil {
		return nil
	}

	e, err := models.FindExecutorForResource(*eventSource.ResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("No executor found for event source %s - skipping execution updates", event.SourceID)
			return nil
		}

		return err
	}

	integration, err := e.FindIntegration()
	if err != nil {
		return err
	}

	executor, err := executors.NewExecutor(integration, e, nil, nil, w.Encryptor)
	if err != nil {
		return err
	}

	status, err := executor.HandleWebhook([]byte(event.Raw))
	if err != nil {
		return err
	}

	if !status.Finished() {
		return nil
	}

	result := models.ResultPassed
	if !status.Successful() {
		result = models.ResultFailed
	}

	executionResource, err := models.FindExecutionResource(status.Id(), *eventSource.ResourceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Infof("No execution resource %s found for source %s - skipping execution update", status.Id(), eventSource.ID)
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
			err = w.handleEventForConnection(tx, event, connection)
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
