package contexts

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// executionBranchEvents resolves which canvas event payload belongs to an upstream
// execution on the current branch (for example, after a For Each fan-out).
type executionBranchEvents struct {
	events                []models.CanvasEvent
	byID                  map[uuid.UUID]models.CanvasEvent
	branchEventIDByParent map[uuid.UUID]uuid.UUID
}

func loadExecutionBranchEvents(
	tx *gorm.DB,
	chainExecutions []models.CanvasNodeExecution,
	executionIDs []uuid.UUID,
) (executionBranchEvents, error) {
	events, err := models.ListCanvasEventsForExecutionsInTransaction(tx, executionIDs)
	if err != nil {
		return executionBranchEvents{}, err
	}

	branchEventIDByParent := branchEventIDByParentExecution(chainExecutions)
	events, err = loadMissingBranchEvents(tx, events, branchEventIDByParent)
	if err != nil {
		return executionBranchEvents{}, err
	}

	return executionBranchEvents{
		events:                events,
		byID:                  indexEventsByID(events),
		branchEventIDByParent: branchEventIDByParent,
	}, nil
}

func (e executionBranchEvents) eventFor(executionID uuid.UUID) (models.CanvasEvent, bool, error) {
	return eventForExecution(executionID, e.events, e.byID, e.branchEventIDByParent)
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

// branchEventIDByParentExecution maps each parent execution to the canvas event that
// routed into the child on the current branch (from child.EventID when
// child.PreviousExecutionID is set).
func branchEventIDByParentExecution(executions []models.CanvasNodeExecution) map[uuid.UUID]uuid.UUID {
	branchEventIDByParent := make(map[uuid.UUID]uuid.UUID, len(executions))
	for _, execution := range executions {
		if execution.PreviousExecutionID == nil || execution.EventID == uuid.Nil {
			continue
		}
		branchEventIDByParent[*execution.PreviousExecutionID] = execution.EventID
	}
	return branchEventIDByParent
}

func indexEventsByID(events []models.CanvasEvent) map[uuid.UUID]models.CanvasEvent {
	byID := make(map[uuid.UUID]models.CanvasEvent, len(events))
	for _, event := range events {
		byID[event.ID] = event
	}
	return byID
}

func loadMissingBranchEvents(
	tx *gorm.DB,
	events []models.CanvasEvent,
	branchEventIDByParent map[uuid.UUID]uuid.UUID,
) ([]models.CanvasEvent, error) {
	byID := indexEventsByID(events)
	for _, eventID := range branchEventIDByParent {
		if eventID == uuid.Nil {
			continue
		}
		if _, ok := byID[eventID]; ok {
			continue
		}

		event, err := models.FindCanvasEventInTransaction(tx, eventID)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
		byID[eventID] = *event
	}
	return events, nil
}

// eventForExecution picks the canvas event whose payload should represent an upstream
// execution on the current branch. When branchEventIDByParent has an entry, it uses the
// child's incoming event ID; otherwise it uses the sole event on that execution.
func eventForExecution(
	executionID uuid.UUID,
	events []models.CanvasEvent,
	eventsByID map[uuid.UUID]models.CanvasEvent,
	branchEventIDByParent map[uuid.UUID]uuid.UUID,
) (models.CanvasEvent, bool, error) {
	if eventID, ok := branchEventIDByParent[executionID]; ok && eventID != uuid.Nil {
		if event, found := eventsByID[eventID]; found {
			return event, true, nil
		}
	}

	var matched []models.CanvasEvent
	for _, event := range events {
		if event.ExecutionID != nil && *event.ExecutionID == executionID {
			matched = append(matched, event)
		}
	}

	switch len(matched) {
	case 0:
		return models.CanvasEvent{}, false, nil
	case 1:
		return matched[0], true, nil
	default:
		return models.CanvasEvent{}, false, fmt.Errorf(
			"execution %s has ambiguous outputs (%d events)",
			executionID,
			len(matched),
		)
	}
}
