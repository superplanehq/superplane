package contexts

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// executionOutputLookup finds the canvas event that represents an upstream
// execution's output for the current run.
//
// Most executions emit one event. Some emit many (Read Memory "One By One",
// For Each, and similar). The run chain records which event each step consumed
// via EventID and PreviousExecutionID on the child execution.
type executionOutputLookup struct {
	eventsByID            map[uuid.UUID]models.CanvasEvent
	eventsByExecutionID   map[uuid.UUID][]models.CanvasEvent
	consumedEventByParent map[uuid.UUID]uuid.UUID
	incomingEventID       *uuid.UUID
}

func newExecutionOutputLookup(
	tx *gorm.DB,
	runChain []models.CanvasNodeExecution,
	executionIDs []uuid.UUID,
	incomingEventID *uuid.UUID,
) (executionOutputLookup, error) {
	consumedEventByParent := consumedEventByParentFromRunChain(runChain)

	events, err := loadOutputEvents(tx, executionIDs, consumedEventByParent)
	if err != nil {
		return executionOutputLookup{}, err
	}

	return executionOutputLookup{
		eventsByID:            indexEventsByID(events),
		eventsByExecutionID:   indexEventsByExecutionID(events),
		consumedEventByParent: consumedEventByParent,
		incomingEventID:       incomingEventID,
	}, nil
}

// outputEvent returns the canvas event whose payload represents executionID
// for this run.
func (l executionOutputLookup) outputEvent(executionID uuid.UUID) (models.CanvasEvent, bool, error) {
	if eventID, ok := l.consumedEventByParent[executionID]; ok && eventID != uuid.Nil {
		if event, found := l.eventsByID[eventID]; found {
			return event, true, nil
		}
	}

	matched := l.eventsByExecutionID[executionID]
	switch len(matched) {
	case 0:
		return models.CanvasEvent{}, false, nil
	case 1:
		return matched[0], true, nil
	default:
		if l.incomingEventID != nil {
			for _, event := range matched {
				if event.ID == *l.incomingEventID {
					return event, true, nil
				}
			}
		}

		return models.CanvasEvent{}, false, fmt.Errorf(
			"execution %s has ambiguous outputs (%d events)",
			executionID,
			len(matched),
		)
	}
}

func loadOutputEvents(
	tx *gorm.DB,
	executionIDs []uuid.UUID,
	consumedEventByParent map[uuid.UUID]uuid.UUID,
) ([]models.CanvasEvent, error) {
	bulkExecutionIDs := make([]uuid.UUID, 0, len(executionIDs))
	for _, executionID := range executionIDs {
		if _, consumed := consumedEventByParent[executionID]; consumed {
			continue
		}
		bulkExecutionIDs = append(bulkExecutionIDs, executionID)
	}

	var events []models.CanvasEvent
	if len(bulkExecutionIDs) > 0 {
		bulkEvents, err := models.ListCanvasEventsForExecutionsInTransaction(tx, bulkExecutionIDs)
		if err != nil {
			return nil, err
		}
		events = append(events, bulkEvents...)
	}

	consumedEventIDs := consumedEventIDsFromMap(consumedEventByParent)
	if len(consumedEventIDs) > 0 {
		consumedEvents, err := models.ListCanvasEventsByIDsInTransaction(tx, consumedEventIDs)
		if err != nil {
			return nil, err
		}
		events = append(events, consumedEvents...)
	}

	return events, nil
}

// consumedEventByParentFromRunChain records which output event each parent
// execution handed to its child in this run chain.
func consumedEventByParentFromRunChain(runChain []models.CanvasNodeExecution) map[uuid.UUID]uuid.UUID {
	consumedEventByParent := make(map[uuid.UUID]uuid.UUID, len(runChain))
	for _, execution := range runChain {
		if execution.PreviousExecutionID == nil || execution.EventID == uuid.Nil {
			continue
		}
		consumedEventByParent[*execution.PreviousExecutionID] = execution.EventID
	}
	return consumedEventByParent
}

func consumedEventIDsFromMap(consumedEventByParent map[uuid.UUID]uuid.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(consumedEventByParent))
	for _, eventID := range consumedEventByParent {
		if eventID == uuid.Nil {
			continue
		}
		ids = append(ids, eventID)
	}
	return ids
}

func indexEventsByID(events []models.CanvasEvent) map[uuid.UUID]models.CanvasEvent {
	byID := make(map[uuid.UUID]models.CanvasEvent, len(events))
	for _, event := range events {
		byID[event.ID] = event
	}
	return byID
}

func indexEventsByExecutionID(events []models.CanvasEvent) map[uuid.UUID][]models.CanvasEvent {
	byExecutionID := make(map[uuid.UUID][]models.CanvasEvent)
	for _, event := range events {
		if event.ExecutionID == nil {
			continue
		}
		executionID := *event.ExecutionID
		byExecutionID[executionID] = append(byExecutionID[executionID], event)
	}
	return byExecutionID
}

func unionExecutionIDs(sets ...[]uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	for _, set := range sets {
		for _, executionID := range set {
			seen[executionID] = struct{}{}
		}
	}

	result := make([]uuid.UUID, 0, len(seen))
	for executionID := range seen {
		result = append(result, executionID)
	}
	return result
}

func executionIDsFromExecutions(executions []models.CanvasNodeExecution) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(executions))
	for _, execution := range executions {
		ids = append(ids, execution.ID)
	}
	return ids
}
