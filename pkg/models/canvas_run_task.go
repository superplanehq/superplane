package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CanvasRunTaskLabel is the task identity shown for a canvas run.
type CanvasRunTaskLabel struct {
	ID    uuid.UUID
	Key   string
	Title string
}

// ResolveCanvasRunTasks returns the task for each run that has an indexed
// factory link. Line executions win, then pull-request runs, then source
// runs, then planning sessions. The nearest ancestor wins inside one link
// type. A missing order is omitted. Closed and archived orders are included.
func ResolveCanvasRunTasks(tx *gorm.DB, canvas *Canvas, runs []CanvasRun) (map[uuid.UUID]CanvasRunTaskLabel, error) {
	labels := map[uuid.UUID]CanvasRunTaskLabel{}
	if len(runs) == 0 {
		return labels, nil
	}

	chains, err := loadRunAncestorChains(tx, runs)
	if err != nil {
		return nil, err
	}

	orderIDs, err := matchRunTaskOrderIDs(tx, canvas, chains)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) == 0 {
		return labels, nil
	}

	return loadCanvasRunTaskLabels(tx, orderIDs)
}

func matchRunTaskOrderIDs(tx *gorm.DB, canvas *Canvas, chains map[uuid.UUID][]uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	pending := make([]uuid.UUID, 0, len(chains))
	for runID := range chains {
		pending = append(pending, runID)
	}
	matched := make(map[uuid.UUID]uuid.UUID, len(chains))

	executions, err := listWorkOrderIDsByExecutionRunIDs(tx, ancestorIDs(chains, pending))
	if err != nil {
		return nil, err
	}
	pending = assignNearestRunTask(chains, pending, executions, matched)

	links, err := FindWorkOrderIDsByLinkedRunIDs(tx, ancestorIDs(chains, pending))
	if err != nil {
		return nil, err
	}
	pending = assignNearestRunTask(chains, pending, links, matched)

	if factoryID, ok := canvasFactoryID(canvas); ok && len(pending) > 0 {
		orders, err := ListWorkOrdersBySourceRunIDs(tx, factoryID, ancestorIDs(chains, pending))
		if err != nil {
			return nil, err
		}
		sourceLinks := make(map[uuid.UUID]uuid.UUID, len(orders))
		for runID, order := range orders {
			sourceLinks[runID] = order.ID
		}
		pending = assignNearestRunTask(chains, pending, sourceLinks, matched)
	}

	sessions, err := listDraftWorkOrderIDsByPlanningRunIDs(tx, ancestorIDs(chains, pending))
	if err != nil {
		return nil, err
	}
	assignNearestRunTask(chains, pending, sessions, matched)
	return matched, nil
}

func CanvasRunTaskLabelsForOrders(tx *gorm.DB, orderIDsByRunID map[uuid.UUID]uuid.UUID) (map[uuid.UUID]CanvasRunTaskLabel, error) {
	if len(orderIDsByRunID) == 0 {
		return map[uuid.UUID]CanvasRunTaskLabel{}, nil
	}
	return loadCanvasRunTaskLabels(tx, orderIDsByRunID)
}

func assignNearestRunTask(chains map[uuid.UUID][]uuid.UUID, pending []uuid.UUID, links map[uuid.UUID]uuid.UUID, matched map[uuid.UUID]uuid.UUID) []uuid.UUID {
	if len(pending) == 0 || len(links) == 0 {
		return pending
	}

	remaining := make([]uuid.UUID, 0, len(pending))
	for _, runID := range pending {
		orderID, ok := nearestRunTaskLink(chains[runID], links)
		if !ok {
			remaining = append(remaining, runID)
			continue
		}
		matched[runID] = orderID
	}
	return remaining
}

func nearestRunTaskLink(chain []uuid.UUID, links map[uuid.UUID]uuid.UUID) (uuid.UUID, bool) {
	for _, runID := range chain {
		orderID, ok := links[runID]
		if ok && orderID != uuid.Nil {
			return orderID, true
		}
	}
	return uuid.Nil, false
}

func ancestorIDs(chains map[uuid.UUID][]uuid.UUID, runIDs []uuid.UUID) []uuid.UUID {
	if len(runIDs) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(runIDs))
	ids := make([]uuid.UUID, 0, len(runIDs))
	for _, runID := range runIDs {
		for _, ancestorID := range chains[runID] {
			if _, ok := seen[ancestorID]; ok {
				continue
			}
			seen[ancestorID] = struct{}{}
			ids = append(ids, ancestorID)
		}
	}
	return ids
}

func canvasFactoryID(canvas *Canvas) (uuid.UUID, bool) {
	if canvas == nil || canvas.FactoryID == nil || *canvas.FactoryID == uuid.Nil {
		return uuid.Nil, false
	}
	return *canvas.FactoryID, true
}

func loadRunAncestorChains(tx *gorm.DB, runs []CanvasRun) (map[uuid.UUID][]uuid.UUID, error) {
	parents := make(map[uuid.UUID]*uuid.UUID, len(runs))
	for i := range runs {
		parents[runs[i].ID] = copyRunID(runs[i].ParentRunID)
	}

	pending := unloadedParentRunIDs(parents)
	for hop := 1; hop < maxFactoryExecutionRunAncestors && len(pending) > 0; hop++ {
		loaded, err := listCanvasRunParentIDs(tx, pending)
		if err != nil {
			return nil, err
		}

		next := make([]uuid.UUID, 0)
		seenNext := map[uuid.UUID]struct{}{}
		for _, runID := range pending {
			parent, ok := loaded[runID]
			if !ok {
				parents[runID] = nil
				continue
			}
			parents[runID] = parent
			if parent == nil {
				continue
			}
			if _, known := parents[*parent]; known {
				continue
			}
			if _, queued := seenNext[*parent]; queued {
				continue
			}
			seenNext[*parent] = struct{}{}
			next = append(next, *parent)
		}
		pending = next
	}

	chains := make(map[uuid.UUID][]uuid.UUID, len(runs))
	for i := range runs {
		chains[runs[i].ID] = walkRunAncestorChain(runs[i].ID, parents)
	}
	return chains, nil
}

func walkRunAncestorChain(start uuid.UUID, parents map[uuid.UUID]*uuid.UUID) []uuid.UUID {
	chain := make([]uuid.UUID, 0, 4)
	seen := make(map[uuid.UUID]struct{}, 4)
	current := start
	for range maxFactoryExecutionRunAncestors {
		if _, visited := seen[current]; visited {
			break
		}
		seen[current] = struct{}{}
		chain = append(chain, current)

		parent, known := parents[current]
		if !known || parent == nil {
			break
		}
		current = *parent
	}
	return chain
}

func unloadedParentRunIDs(parents map[uuid.UUID]*uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	ids := make([]uuid.UUID, 0)
	for _, parent := range parents {
		if parent == nil {
			continue
		}
		if _, loaded := parents[*parent]; loaded {
			continue
		}
		if _, queued := seen[*parent]; queued {
			continue
		}
		seen[*parent] = struct{}{}
		ids = append(ids, *parent)
	}
	return ids
}

type canvasRunParentRow struct {
	ID          uuid.UUID
	ParentRunID *uuid.UUID
}

func listCanvasRunParentIDs(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]*uuid.UUID, error) {
	parents := make(map[uuid.UUID]*uuid.UUID, len(runIDs))
	if len(runIDs) == 0 {
		return parents, nil
	}

	var rows []canvasRunParentRow
	err := tx.Model(&CanvasRun{}).
		Select("id", "parent_run_id").
		Where("id IN ?", runIDs).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load run parents: %w", err)
	}

	for _, row := range rows {
		parents[row.ID] = copyRunID(row.ParentRunID)
	}
	return parents, nil
}

func copyRunID(id *uuid.UUID) *uuid.UUID {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	copied := *id
	return &copied
}

type executionRunOrderRow struct {
	RunID       uuid.UUID
	WorkOrderID uuid.UUID
}

func listWorkOrderIDsByExecutionRunIDs(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	orders := map[uuid.UUID]uuid.UUID{}
	if len(runIDs) == 0 {
		return orders, nil
	}

	var rows []executionRunOrderRow
	err := tx.Model(&FactoryWorkOrderExecution{}).
		Select("run_id", "work_order_id").
		Where("run_id IN ?", runIDs).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load line executions for runs: %w", err)
	}
	for _, row := range rows {
		if row.RunID == uuid.Nil || row.WorkOrderID == uuid.Nil {
			continue
		}
		orders[row.RunID] = row.WorkOrderID
	}
	return orders, nil
}

type planningRunOrderRow struct {
	CanvasRunID      uuid.UUID
	DraftWorkOrderID uuid.UUID
}

func listDraftWorkOrderIDsByPlanningRunIDs(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	orders := map[uuid.UUID]uuid.UUID{}
	if len(runIDs) == 0 {
		return orders, nil
	}

	var rows []planningRunOrderRow
	err := tx.Model(&FactoryPlanningSession{}).
		Select("canvas_run_id", "draft_work_order_id").
		Where("canvas_run_id IN ?", runIDs).
		Where("draft_work_order_id IS NOT NULL").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load planning sessions for runs: %w", err)
	}
	for _, row := range rows {
		if row.CanvasRunID == uuid.Nil || row.DraftWorkOrderID == uuid.Nil {
			continue
		}
		orders[row.CanvasRunID] = row.DraftWorkOrderID
	}

	var refinements []FactoryWorkOrderRefinementRun
	if err := tx.Where("run_id IN ?", runIDs).Find(&refinements).Error; err != nil {
		return nil, fmt.Errorf("load refinement runs: %w", err)
	}
	for _, link := range refinements {
		orders[link.RunID] = link.WorkOrderID
	}
	return orders, nil
}

type canvasRunTaskOrderRow struct {
	ID        uuid.UUID
	Number    int64
	Title     string
	FactoryID uuid.UUID
}

type canvasRunTaskFactoryRow struct {
	ID  uuid.UUID
	Key string
}

func loadCanvasRunTaskLabels(tx *gorm.DB, orderIDsByRunID map[uuid.UUID]uuid.UUID) (map[uuid.UUID]CanvasRunTaskLabel, error) {
	orderIDs := uniqueRunTaskOrderIDs(orderIDsByRunID)
	var orders []canvasRunTaskOrderRow
	err := tx.Model(&FactoryWorkOrder{}).
		Select("id", "number", "title", "factory_id").
		Where("id IN ?", orderIDs).
		Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("load run task orders: %w", err)
	}

	ordersByID := make(map[uuid.UUID]canvasRunTaskOrderRow, len(orders))
	factoryIDs := make([]uuid.UUID, 0, len(orders))
	seenFactories := map[uuid.UUID]struct{}{}
	for _, order := range orders {
		ordersByID[order.ID] = order
		if order.FactoryID == uuid.Nil {
			continue
		}
		if _, seen := seenFactories[order.FactoryID]; seen {
			continue
		}
		seenFactories[order.FactoryID] = struct{}{}
		factoryIDs = append(factoryIDs, order.FactoryID)
	}

	keys, err := loadCanvasRunTaskFactoryKeys(tx, factoryIDs)
	if err != nil {
		return nil, err
	}

	labels := make(map[uuid.UUID]CanvasRunTaskLabel, len(orderIDsByRunID))
	for runID, orderID := range orderIDsByRunID {
		order, ok := ordersByID[orderID]
		if !ok {
			continue
		}
		label := CanvasRunTaskLabel{ID: order.ID, Title: order.Title}
		if key, ok := keys[order.FactoryID]; ok {
			factory := Factory{Key: key}
			label.Key = factory.WorkOrderKey(order.Number)
		}
		labels[runID] = label
	}
	return labels, nil
}

func uniqueRunTaskOrderIDs(orderIDsByRunID map[uuid.UUID]uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(orderIDsByRunID))
	ids := make([]uuid.UUID, 0, len(orderIDsByRunID))
	for _, orderID := range orderIDsByRunID {
		if orderID == uuid.Nil {
			continue
		}
		if _, ok := seen[orderID]; ok {
			continue
		}
		seen[orderID] = struct{}{}
		ids = append(ids, orderID)
	}
	return ids
}

func loadCanvasRunTaskFactoryKeys(tx *gorm.DB, factoryIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	keys := map[uuid.UUID]string{}
	if len(factoryIDs) == 0 {
		return keys, nil
	}

	var rows []canvasRunTaskFactoryRow
	err := tx.Model(&Factory{}).
		Select("id", "key").
		Where("id IN ?", factoryIDs).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load run task factory keys: %w", err)
	}
	for _, row := range rows {
		keys[row.ID] = row.Key
	}
	return keys, nil
}
