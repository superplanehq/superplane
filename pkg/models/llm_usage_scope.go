package models

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// usageScope is the factory billing context for one LLM call.
type usageScope struct {
	OrganizationID       uuid.UUID
	FactoryID            uuid.UUID
	WorkOrderID          *uuid.UUID
	LineID               *uuid.UUID
	LineDispatchID       *uuid.UUID
	WorkOrderExecutionID *uuid.UUID
	execution            *FactoryWorkOrderExecution
}

func scopeFromLineExecution(execution *FactoryWorkOrderExecution) *usageScope {
	workOrderID := execution.WorkOrderID
	lineID := execution.LineID
	dispatchID := execution.LineDispatchID
	executionID := execution.ID
	return &usageScope{
		OrganizationID:       execution.OrganizationID,
		FactoryID:            execution.FactoryID,
		WorkOrderID:          &workOrderID,
		LineID:               &lineID,
		LineDispatchID:       &dispatchID,
		WorkOrderExecutionID: &executionID,
		execution:            execution,
	}
}

// resolveUsageScope finds factory billing fields for a canvas run. Line
// executions win. Otherwise the canvas must belong to a factory. The work
// order comes from a linked PR run, a source_run_id, or an onWorkOrder
// root event. Returns nil when the run is not a factory canvas.
func resolveUsageScope(tx *gorm.DB, runID uuid.UUID) (*usageScope, error) {
	execution, err := FindWorkOrderExecutionForRun(tx, runID)
	if err == nil {
		return scopeFromLineExecution(execution), nil
	}
	if !errors.Is(err, ErrFactoryWorkOrderExecutionNotFound) {
		return nil, err
	}

	canvas, err := findFactoryCanvasForRun(tx, runID)
	if err != nil || canvas == nil {
		return nil, err
	}

	workOrderID, err := findWorkOrderIDForFactoryRun(tx, runID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	return &usageScope{
		OrganizationID: canvas.OrganizationID,
		FactoryID:      *canvas.FactoryID,
		WorkOrderID:    workOrderID,
	}, nil
}

func findFactoryCanvasForRun(tx *gorm.DB, runID uuid.UUID) (*Canvas, error) {
	seen := make(map[uuid.UUID]struct{}, 8)
	current := runID
	for range maxFactoryExecutionRunAncestors {
		if _, visited := seen[current]; visited {
			return nil, nil
		}
		seen[current] = struct{}{}

		run, err := FindUnscopedCanvasRun(tx, current)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}

		canvas, err := FindCanvasWithoutOrgScopeInTransaction(tx, run.WorkflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		if canvas.FactoryID != nil && *canvas.FactoryID != uuid.Nil {
			return canvas, nil
		}

		if run.ParentRunID == nil || *run.ParentRunID == uuid.Nil {
			return nil, nil
		}
		current = *run.ParentRunID
	}
	return nil, nil
}

func findWorkOrderIDForFactoryRun(tx *gorm.DB, runID, factoryID uuid.UUID) (*uuid.UUID, error) {
	runIDs, err := ancestorRunIDs(tx, runID)
	if err != nil || len(runIDs) == 0 {
		return nil, err
	}

	linked, err := FindWorkOrderIDsByLinkedRunIDs(tx, runIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range runIDs {
		if workOrderID, ok := linked[id]; ok {
			return &workOrderID, nil
		}
	}

	orders, err := ListWorkOrdersBySourceRunIDs(tx, factoryID, runIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range runIDs {
		if order, ok := orders[id]; ok {
			workOrderID := order.ID
			return &workOrderID, nil
		}
	}

	for _, id := range runIDs {
		workOrderID, err := workOrderIDFromRunEvents(tx, id)
		if err != nil {
			return nil, err
		}
		if workOrderID != nil {
			return workOrderID, nil
		}
	}
	return nil, nil
}

func ancestorRunIDs(tx *gorm.DB, runID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, 8)
	seen := make(map[uuid.UUID]struct{}, 8)
	current := runID
	for range maxFactoryExecutionRunAncestors {
		if _, visited := seen[current]; visited {
			break
		}
		seen[current] = struct{}{}

		run, err := FindUnscopedCanvasRun(tx, current)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, err
		}
		ids = append(ids, run.ID)
		if run.ParentRunID == nil || *run.ParentRunID == uuid.Nil {
			break
		}
		current = *run.ParentRunID
	}
	return ids, nil
}

func workOrderIDFromRunEvents(tx *gorm.DB, runID uuid.UUID) (*uuid.UUID, error) {
	var event CanvasEvent
	err := tx.Where("run_id = ?", runID).Order("created_at ASC").First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return workOrderIDFromEventData(event.Data.Data()), nil
}

func workOrderIDFromEventData(data any) *uuid.UUID {
	payload, ok := objectMap(RootEventSourcePayload(data))
	if !ok {
		return nil
	}

	workOrder, ok := objectMap(payload["workOrder"])
	if !ok {
		workOrder, ok = objectMap(payload["work_order"])
	}
	if !ok {
		return nil
	}

	raw, ok := workOrder["id"].(string)
	if !ok || raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

func objectMap(value any) (map[string]any, bool) {
	switch current := value.(type) {
	case map[string]any:
		return current, true
	case json.RawMessage:
		var decoded map[string]any
		if err := json.Unmarshal(current, &decoded); err != nil {
			return nil, false
		}
		return decoded, true
	default:
		return nil, false
	}
}
