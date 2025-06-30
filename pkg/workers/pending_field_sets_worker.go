package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

type PendingFieldSetsWorker struct {
	nowFunc func() time.Time
}

func NewPendingFieldSetsWorker(nowFunc func() time.Time) (*PendingFieldSetsWorker, error) {
	if nowFunc == nil {
		return nil, fmt.Errorf("nowFunc is required")
	}

	return &PendingFieldSetsWorker{nowFunc: nowFunc}, nil
}

func (w *PendingFieldSetsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing events: %v", err)
		}

		time.Sleep(time.Minute)
	}
}

func (w *PendingFieldSetsWorker) Tick() error {
	fieldSets, err := models.ListPendingConnectionGroupFieldSets()
	if err != nil {
		return err
	}

	for _, fieldSet := range fieldSets {
		err := w.ProcessFieldSet(fieldSet)
		if err != nil {
			log.Errorf("Error processing field set %s: %v", fieldSet.ID, err)
		}
	}

	return nil
}

func (w *PendingFieldSetsWorker) ProcessFieldSet(fieldSet models.ConnectionGroupFieldSet) error {
	connectionGroup, err := models.FindConnectionGroupByID(database.Conn(), fieldSet.ConnectionGroupID)
	if err != nil {
		return fmt.Errorf("error finding connection group %s: %v", fieldSet.ConnectionGroupID, err)
	}

	connectionGroupSpec := connectionGroup.Spec.Data()

	//
	// If we still haven't hit the timeout, we do nothing.
	//
	if !fieldSet.IsTimedOut(w.nowFunc()) {
		log.Infof("Field set %s for %s has not timed out - skipping", fieldSet.String(), connectionGroup.Name)
		return nil
	}

	missingConnections, err := fieldSet.MissingConnections(database.Conn(), connectionGroup)
	if err != nil {
		return err
	}

	switch connectionGroupSpec.TimeoutBehavior {
	case models.ConnectionGroupTimeoutBehaviorEmit:
		log.Infof("Field set %s for %s has timed out - processing", fieldSet.String(), connectionGroup.Name)

		return connectionGroup.Emit(&fieldSet, models.ConnectionGroupFieldSetStateReasonTimeout, missingConnections)

	case models.ConnectionGroupTimeoutBehaviorDrop:
		log.Infof("Field set %s for %s timed out - discarding", fieldSet.String(), connectionGroup.Name)
		return fieldSet.UpdateState(
			database.Conn(),
			models.ConnectionGroupFieldSetStateDiscarded,
			models.ConnectionGroupFieldSetStateReasonTimeout,
		)

	default:
		return fmt.Errorf("invalid timeout behavior: %s", connectionGroupSpec.TimeoutBehavior)
	}
}
