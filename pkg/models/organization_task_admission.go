package models

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const organizationTaskAdmissionBatchSize = 100

// ErrOrganizationTaskCapacity means this task cannot start without exceeding
// the organization limit or passing an older eligible task.
var ErrOrganizationTaskCapacity = errors.New("organization task capacity reached")

// ResolveOrganizationParallelTaskLimit uses the effective plan, including trial
// and subscription expiry. A nil limit means unlimited organization capacity.
func ResolveOrganizationParallelTaskLimit(tx *gorm.DB, orgID uuid.UUID) (*int, error) {
	if !BillingEnabled() {
		return nil, nil
	}
	plan, err := ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	return plan.Catalog.ParallelTaskLimit, nil
}

// LockOrganizationTaskAdmission must precede row locks and writes in the outer
// transaction. Reacquiring it in nested model calls is safe. The namespace keeps
// these advisory keys separate from other application locks.
func LockOrganizationTaskAdmission(tx *gorm.DB, orgID uuid.UUID) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", organizationAdmissionKey(orgID)).Error
}

func TryLockOrganizationTaskAdmission(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	var locked bool
	err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", organizationAdmissionKey(orgID)).Scan(&locked).Error
	return locked, err
}

// LockRunOrganizationTaskAdmission also covers automation callbacks that close
// orders. Resolve without row locks, then lock the organization before the run.
func LockRunOrganizationTaskAdmission(tx *gorm.DB, runID uuid.UUID) error {
	var owner struct{ OrganizationID uuid.UUID }
	err := tx.Table("workflow_runs AS r").Select("c.organization_id").
		Joins("JOIN workflows c ON c.id = r.workflow_id").Where("r.id = ?", runID).Scan(&owner).Error
	if err != nil || owner.OrganizationID == uuid.Nil {
		return err
	}
	return LockOrganizationTaskAdmission(tx, owner.OrganizationID)
}

// LockTaskStatusExecutionAdmission locks before the execution row because this
// action can close an order and remove its queued work.
func LockTaskStatusExecutionAdmission(tx *gorm.DB, executionID uuid.UUID) error {
	var owner struct{ OrganizationID uuid.UUID }
	err := tx.Table("workflow_node_executions AS e").Select("c.organization_id").
		Joins("JOIN workflows c ON c.id = e.workflow_id").
		Joins("JOIN workflow_nodes n ON n.workflow_id = e.workflow_id AND n.node_id = e.node_id").
		Where("e.id = ? AND n.ref->'component'->>'name' = ?", executionID, "updateWorkOrderStatus").Scan(&owner).Error
	if err != nil || owner.OrganizationID == uuid.Nil {
		return err
	}
	return LockOrganizationTaskAdmission(tx, owner.OrganizationID)
}

// LockTaskExecutionAdmission locks only components that change task admission,
// before the execution row. Other components can wait on external services
// without blocking admission throughout the organization.
func LockTaskExecutionAdmission(tx *gorm.DB, executionID uuid.UUID) error {
	var owner struct{ OrganizationID uuid.UUID }
	err := tx.Table("workflow_node_executions AS e").Select("c.organization_id").
		Joins("JOIN workflows c ON c.id = e.workflow_id AND c.factory_id IS NOT NULL").
		Joins("JOIN workflow_nodes n ON n.workflow_id = e.workflow_id AND n.node_id = e.node_id").
		Where("e.id = ? AND COALESCE(NULLIF(BTRIM(e.metadata->>'frozenComponent'), ''), n.ref->'component'->>'name') IN ?", executionID,
			[]string{"updateWorkOrderStatus", "addPullRequestActivity"}).Scan(&owner).Error
	if err != nil || owner.OrganizationID == uuid.Nil {
		return err
	}
	return LockOrganizationTaskAdmission(tx, owner.OrganizationID)
}

// CheckTaskRootEventAdmission applies organization capacity to a factory root
// event that names an existing draft or open task. Events without that task
// association continue. A capacity error leaves the event pending.
func CheckTaskRootEventAdmission(tx *gorm.DB, canvas *Canvas, event *CanvasEvent) error {
	if canvas == nil || event == nil || canvas.FactoryID == nil {
		return nil
	}
	workOrderID := workOrderIDFromEventData(event.Data.Data())
	if workOrderID == nil {
		return nil
	}
	var order FactoryWorkOrder
	err := tx.Where("id = ? AND organization_id = ? AND factory_id = ?", *workOrderID, canvas.OrganizationID, *canvas.FactoryID).
		First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !order.IsDispatchable() {
		return nil
	}
	return checkOrganizationTaskAdmission(tx, canvas.OrganizationID, order.ID)
}

func ListQueuedTaskOrganizationIDs(tx *gorm.DB) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Model(&FactoryWorkOrderQueueItem{}).Distinct().Order("organization_id").Pluck("organization_id", &ids).Error
	return ids, err
}

// AdmitQueuedForOrganization selects globally by enqueue time and ID, skipping
// locally full steps. Each pass inspects at most one candidate batch. At full
// capacity, only active tasks can continue. Results include removed queue items.
// The organization lock remains outside candidate savepoints.
func AdmitQueuedForOrganization(tx *gorm.DB, orgID uuid.UUID) ([]*FactoryLineStepResult, error) {
	if err := LockOrganizationTaskAdmission(tx, orgID); err != nil {
		return nil, err
	}
	slots, err := organizationAvailableTaskSlots(tx, orgID)
	if err != nil {
		return nil, err
	}
	continuable, err := continuableOrganizationTaskIDs(tx, orgID)
	if err != nil {
		return nil, err
	}
	var items []FactoryWorkOrderQueueItem
	candidates := eligibleQueuedTasks(tx, orgID)
	if slots != nil && *slots <= 0 {
		if len(continuable) == 0 {
			return nil, nil
		}
		candidates = candidates.Where("work_order_id IN ?", continuable)
	}
	if err := candidates.Limit(organizationTaskAdmissionBatchSize).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	var head *OrganizationQueuedTask
	if slots == nil || *slots > 0 {
		head, err = nextQueuedOrganizationTask(tx, orgID)
		if err != nil {
			return nil, err
		}
	}
	active, err := ListActiveOrganizationTaskIDs(tx, orgID)
	if err != nil {
		return nil, err
	}
	var admitted []*FactoryLineStepResult
	for i := range items {
		item := &items[i]
		continuing := slices.Contains(continuable, item.WorkOrderID)
		if !continuing && slots != nil && *slots <= 0 {
			continue
		}
		// A pending root event is not a line item. Leave newer line items
		// queued until the event router admits that older task.
		if !continuing && head != nil && head.RunID != uuid.Nil && head.WorkOrderID != item.WorkOrderID && !queuedLineItemBefore(item, head) {
			continue
		}
		result, err := admitQueuedTask(tx, item)
		if err != nil {
			return nil, fmt.Errorf("organization %s queue item %s: %w", orgID, item.ID, err)
		}
		if result != nil {
			admitted = append(admitted, result)
			if slots != nil && result.Run != nil && !slices.Contains(active, item.WorkOrderID) {
				*slots--
				active = append(active, item.WorkOrderID)
				continuable = append(continuable, item.WorkOrderID)
			}
		}
	}
	return admitted, nil
}

// Replacements must wait for their old line runs to stop. Other active tasks
// can continue line work without reserving another organization slot.
func continuableOrganizationTaskIDs(tx *gorm.DB, orgID uuid.UUID) ([]uuid.UUID, error) {
	active, err := ListActiveOrganizationTaskIDs(tx, orgID)
	if err != nil || len(active) == 0 {
		return active, err
	}
	var ids []uuid.UUID
	err = tx.Model(&FactoryWorkOrder{}).Where("id IN ?", active).
		Where(`NOT EXISTS (SELECT 1 FROM factory_work_order_executions e
			JOIN workflow_runs r ON r.id = e.run_id
			WHERE e.work_order_id = factory_work_orders.id AND r.state = ?)`, CanvasRunStateCancelling).
		Pluck("id", &ids).Error
	return ids, err
}

func checkOrganizationTaskAdmission(tx *gorm.DB, orgID, workOrderID uuid.UUID) error {
	if workOrderID == uuid.Nil {
		return nil
	}
	active, err := ListActiveOrganizationTaskIDs(tx, orgID)
	if err != nil {
		return err
	}
	if slices.Contains(active, workOrderID) {
		return nil
	}
	limit, err := ResolveOrganizationParallelTaskLimit(tx, orgID)
	if err != nil {
		return err
	}
	if limit != nil && len(active) >= *limit {
		return ErrOrganizationTaskCapacity
	}
	next, err := nextQueuedOrganizationTask(tx, orgID)
	if err != nil {
		return err
	}
	if next != nil && next.WorkOrderID != workOrderID {
		return ErrOrganizationTaskCapacity
	}
	return nil
}

func queuedLineItemBefore(item *FactoryWorkOrderQueueItem, head *OrganizationQueuedTask) bool {
	if order := item.CreatedAt.Compare(head.CreatedAt); order != 0 {
		return order < 0
	}
	return item.ID.String() < head.ID.String()
}

func organizationAtTaskCapacity(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	slots, err := organizationAvailableTaskSlots(tx, orgID)
	return slots != nil && *slots <= 0, err
}

func organizationAvailableTaskSlots(tx *gorm.DB, orgID uuid.UUID) (*int64, error) {
	limit, err := ResolveOrganizationParallelTaskLimit(tx, orgID)
	if err != nil || limit == nil {
		return nil, err
	}
	active, err := ListActiveOrganizationTaskIDs(tx, orgID)
	if err != nil {
		return nil, err
	}
	slots := int64(*limit - len(active))
	return &slots, nil
}

// Finished execution history does not release a reservation while its replaced
// run is still stopping. Count that run until finalization confirms it stopped.
func activeFactoryTaskReservations(tx *gorm.DB) *gorm.DB {
	return tx.Model(&FactoryWorkOrderExecution{}).Where(`status IN ? OR EXISTS (
		SELECT 1 FROM workflow_runs r WHERE r.id = factory_work_order_executions.run_id AND r.state <> ?
	)`, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning}, CanvasRunStateFinished)
}

// A candidate load does not lock rows: the outer organization lock serializes
// admission, replacement, and order closure. Missing work is stale, not retryable.
func queuedTaskCandidate(tx *gorm.DB, item *FactoryWorkOrderQueueItem) (*FactoryWorkOrderLineDispatch, *FactoryWorkOrder, *FactoryLine, error) {
	dispatch, err := FindWorkOrderLineDispatch(tx, item.LineDispatchID)
	if errors.Is(err, ErrFactoryWorkOrderLineDispatchNotFound) {
		return nil, nil, nil, nil
	}
	if err != nil || !dispatch.IsActive() {
		return dispatch, nil, nil, err
	}
	workspace, err := FindFactory(tx, item.OrganizationID, item.FactoryID)
	if errors.Is(err, ErrFactoryNotFound) {
		return dispatch, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}
	order, err := workspace.FindWorkOrder(tx, item.WorkOrderID)
	if errors.Is(err, ErrFactoryWorkOrderNotFound) {
		return dispatch, nil, nil, nil
	}
	if err != nil || !order.IsDispatchable() {
		return dispatch, nil, nil, err
	}
	line, err := workspace.FindLine(tx, item.LineID)
	if errors.Is(err, ErrFactoryLineNotFound) {
		return dispatch, order, nil, nil
	}
	return dispatch, order, line, err
}

func admitQueuedTask(tx *gorm.DB, item *FactoryWorkOrderQueueItem) (*FactoryLineStepResult, error) {
	dispatch, order, line, err := queuedTaskCandidate(tx, item)
	if err != nil {
		return nil, err
	}
	if dispatch == nil || order == nil || !dispatch.IsActive() {
		return &FactoryLineStepResult{QueueItem: item}, discardQueuedTask(tx, item, dispatch, CanvasRunResultCancelled)
	}
	if line == nil {
		return failQueuedTask(tx, item, dispatch, ErrFactoryLineNotFound)
	}
	if item.StepIndex < 0 || item.StepIndex >= len(dispatch.Steps) {
		return failQueuedTask(tx, item, dispatch, fmt.Errorf("step index %d out of range", item.StepIndex))
	}
	active, err := countActiveFactoryStepExecutions(tx, item.LineID, item.StepIndex)
	if err != nil {
		return nil, err
	}
	if active >= int64(admissionStep(line, dispatch, item.StepIndex).EffectiveMaxParallelism()) {
		return nil, nil
	}
	var result *FactoryLineStepResult
	// GORM creates a savepoint for this nested transaction. Run creation and
	// deletion roll back together before an invalid candidate is cleaned up.
	err = tx.Transaction(func(candidate *gorm.DB) error {
		if err := item.Delete(candidate); err != nil {
			return err
		}
		var err error
		result, err = dispatch.StartStep(candidate, order, item.StepIndex)
		if err != nil || order.IsOpen() {
			return err
		}
		// A queued initial dispatch stays draft until its run exists.
		return order.TransitionOnDispatch(candidate, nil)
	})
	if errors.Is(err, errQueuedTaskUnstartable) {
		return failQueuedTask(tx, item, dispatch, err)
	}
	return result, err
}

// Filter locally full steps before the bounded candidate batch, so a long
// blocked queue cannot hide eligible work in another workspace.
func eligibleQueuedTasks(tx *gorm.DB, orgID uuid.UUID) *gorm.DB {
	return tx.Model(&FactoryWorkOrderQueueItem{}).
		Where("organization_id = ?", orgID).
		Where(`NOT EXISTS (
			SELECT 1 FROM factory_lines l
			JOIN factory_work_order_line_dispatches d ON d.id = factory_work_order_queue_items.line_dispatch_id
			WHERE l.id = factory_work_order_queue_items.line_id
			AND (SELECT COUNT(*) FROM factory_work_order_executions e
				WHERE e.line_id = l.id AND e.step_index = factory_work_order_queue_items.step_index
				AND (e.status IN ? OR EXISTS (SELECT 1 FROM workflow_runs r WHERE r.id = e.run_id AND r.state <> ?)))
			>= COALESCE(NULLIF(GREATEST((COALESCE(l.steps->factory_work_order_queue_items.step_index,
				d.steps->factory_work_order_queue_items.step_index)->>'max_parallelism')::int, 0), 0), ?)
		)`, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning}, CanvasRunStateFinished, DefaultFactoryLineStepMaxParallelism).
		Order("created_at, id")
}

func failQueuedTask(tx *gorm.DB, item *FactoryWorkOrderQueueItem, dispatch *FactoryWorkOrderLineDispatch, cause error) (*FactoryLineStepResult, error) {
	log.WithError(cause).WithFields(log.Fields{"organization_id": item.OrganizationID, "queue_item_id": item.ID, "factory_id": item.FactoryID, "order_id": item.WorkOrderID}).Warn("Discarding unstartable queued task")
	return &FactoryLineStepResult{QueueItem: item}, discardQueuedTask(tx, item, dispatch, CanvasRunResultFailed)
}

func discardQueuedTask(tx *gorm.DB, item *FactoryWorkOrderQueueItem, dispatch *FactoryWorkOrderLineDispatch, result string) error {
	if err := item.Delete(tx); err != nil {
		return err
	}
	if dispatch == nil || !dispatch.IsActive() {
		return nil
	}
	open, err := dispatch.HasOpenWork(tx)
	if err != nil || open {
		return err
	}
	return dispatch.Finish(tx, result)
}

func hasEligibleQueuedTask(tx *gorm.DB, orgID uuid.UUID) (bool, error) {
	roots, err := listQueuedRootTasks(tx, orgID, 1)
	if err != nil {
		return false, err
	}
	if len(roots) > 0 {
		return true, nil
	}
	var items []FactoryWorkOrderQueueItem
	if err := eligibleQueuedTasks(tx, orgID).Limit(organizationTaskAdmissionBatchSize).Find(&items).Error; err != nil {
		return false, err
	}
	for i := range items {
		dispatch, order, line, err := queuedTaskCandidate(tx, &items[i])
		if err != nil {
			return false, err
		}
		if dispatch == nil || order == nil || line == nil || items[i].StepIndex < 0 || items[i].StepIndex >= len(dispatch.Steps) {
			continue
		}
		active, err := countActiveFactoryStepExecutions(tx, line.ID, items[i].StepIndex)
		if err != nil {
			return false, err
		}
		if active < int64(admissionStep(line, dispatch, items[i].StepIndex).EffectiveMaxParallelism()) {
			return true, nil
		}
	}
	return false, nil
}

var errQueuedTaskUnstartable = errors.New("queued task cannot start")

func queuedTaskLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", errQueuedTaskUnstartable, err)
	}
	return err
}

func organizationAdmissionKey(orgID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("organization-task-admission:"), orgID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
