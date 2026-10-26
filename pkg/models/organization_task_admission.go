package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const organizationTaskAdmissionBatchSize = 100

// ErrWorkOrderBusy means earlier work for this task must stop or finish merging.
var ErrWorkOrderBusy = errors.New("work order has stopping work or a merge in progress")

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

// WorkOrderMergeAdvisoryKey identifies the session lock held while a merge
// waits for the Git host. It is distinct from the organization admission lock.
func WorkOrderMergeAdvisoryKey(workOrderID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("work-order-merge:"), workOrderID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

// AcquireWorkOrderMergeLock blocks until this task's merge lock is free.
// conn must be a dedicated connection, not one from the application pool.
// The lock survives until ReleaseWorkOrderMergeLock or the connection closes.
func AcquireWorkOrderMergeLock(ctx context.Context, conn *sql.Conn, workOrderID uuid.UUID) error {
	_, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", WorkOrderMergeAdvisoryKey(workOrderID))
	return err
}

// ReleaseWorkOrderMergeLock drops the session lock acquired by AcquireWorkOrderMergeLock.
func ReleaseWorkOrderMergeLock(ctx context.Context, conn *sql.Conn, workOrderID uuid.UUID) error {
	_, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", WorkOrderMergeAdvisoryKey(workOrderID))
	return err
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

// Components that take the organization lock during execution. The lock must
// already be held before the execution row, matching cancellation. Other
// component calls must not hold it while they wait on the network.
var taskAdmissionLockComponents = []string{"updateWorkOrderStatus", "addPullRequestActivity"}

// LockTaskExecutionAdmission locks before the execution row only when this
// execution can change task admission or takes the organization lock itself.
// addPullRequestActivity and its hook call CreateActivity after the row lock.
// Other component calls must not hold the organization lock while they wait
// on the network.
func LockTaskExecutionAdmission(tx *gorm.DB, executionID uuid.UUID) error {
	var owner struct{ OrganizationID uuid.UUID }
	err := tx.Table("workflow_node_executions AS e").Select("c.organization_id").
		Joins("JOIN workflows c ON c.id = e.workflow_id").
		Joins("JOIN workflow_nodes n ON n.workflow_id = e.workflow_id AND n.node_id = e.node_id").
		Where("e.id = ? AND COALESCE(NULLIF(BTRIM(e.metadata->>'frozenComponent'), ''), n.ref->'component'->>'name') IN ?", executionID, taskAdmissionLockComponents).
		Scan(&owner).Error
	if err != nil || owner.OrganizationID == uuid.Nil {
		return err
	}
	return LockOrganizationTaskAdmission(tx, owner.OrganizationID)
}

// CheckTaskRootEventReady prevents replacement work while the earlier run
// subtree is stopping or the work order is merging.
func CheckTaskRootEventReady(tx *gorm.DB, canvas *Canvas, event *CanvasEvent) error {
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
	return checkWorkOrderReady(tx, canvas.OrganizationID, order.ID)
}

func ListQueuedTaskOrganizationIDs(tx *gorm.DB) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Model(&FactoryWorkOrderQueueItem{}).Distinct().Order("organization_id").Pluck("organization_id", &ids).Error
	return ids, err
}

// AdmitQueuedForOrganization drains eligible line steps. Organization runner
// capacity is enforced later, when each runner job receives a reservation.
func AdmitQueuedForOrganization(tx *gorm.DB, orgID uuid.UUID) ([]*FactoryLineStepResult, error) {
	if err := LockOrganizationTaskAdmission(tx, orgID); err != nil {
		return nil, err
	}
	eligible, err := eligibleQueuedTasks(tx, orgID)
	if err != nil {
		return nil, err
	}
	var items []FactoryWorkOrderQueueItem
	if err := eligible.Limit(organizationTaskAdmissionBatchSize).Find(&items).Error; err != nil {
		return nil, err
	}
	var admitted []*FactoryLineStepResult
	for i := range items {
		result, err := admitQueuedTask(tx, &items[i])
		if err != nil {
			return nil, fmt.Errorf("organization %s queue item %s: %w", orgID, items[i].ID, err)
		}
		if result != nil {
			admitted = append(admitted, result)
		}
	}
	return admitted, nil
}

// checkWorkOrderReady preserves replacement and merge safety independently of
// runner capacity. It does not admit tasks or count task slots.
func checkWorkOrderReady(tx *gorm.DB, orgID, workOrderID uuid.UUID) error {
	if workOrderID == uuid.Nil {
		return nil
	}
	merging, err := workOrderMergeInProgress(tx, workOrderID)
	if err != nil {
		return err
	}
	if merging {
		return ErrWorkOrderBusy
	}
	stopping, err := organizationTaskIsStopping(tx, orgID, workOrderID)
	if err != nil {
		return err
	}
	if stopping {
		return ErrWorkOrderBusy
	}
	return nil
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
	// Take the merge lock only for a task this pass will start. A full step
	// stays queued without blocking that task's merge.
	reserved, err := workOrderMergeInProgress(tx, item.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if reserved {
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
func eligibleQueuedTasks(tx *gorm.DB, orgID uuid.UUID) (*gorm.DB, error) {
	reserved, err := mergeReservedWorkOrderIDs(tx, orgID)
	if err != nil {
		return nil, err
	}
	query := tx.Model(&FactoryWorkOrderQueueItem{}).
		Where("organization_id = ?", orgID).
		Where("work_order_id NOT IN (?)", stoppingOrganizationTasks(tx, orgID))
	if len(reserved) > 0 {
		query = query.Where("work_order_id NOT IN ?", reserved)
	}
	return query.
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
		Order("created_at, id"), nil
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

var errQueuedTaskUnstartable = errors.New("queued task cannot start")

func queuedTaskLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", errQueuedTaskUnstartable, err)
	}
	return err
}

// workOrderMergeInProgress reports a merge waiting on the Git host. A false
// result holds this transaction's advisory lock until commit so the merge
// cannot start while this transaction starts work.
func workOrderMergeInProgress(tx *gorm.DB, workOrderID uuid.UUID) (bool, error) {
	var locked bool
	err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", WorkOrderMergeAdvisoryKey(workOrderID)).Scan(&locked).Error
	return !locked, err
}

// mergeReservedWorkOrderIDs lists queued tasks whose merge lock is held by
// another session. This read does not take the lock, so a waiting merge
// cannot become the head of the queue.
func mergeReservedWorkOrderIDs(tx *gorm.DB, orgID uuid.UUID) ([]uuid.UUID, error) {
	var locks []struct {
		ClassID int64
		ObjID   int64
	}
	err := tx.Raw(`
		SELECT classid::bigint AS class_id, objid::bigint AS obj_id
		FROM pg_locks
		WHERE locktype = 'advisory' AND objsubid = 1 AND granted AND pid <> pg_backend_pid()
	`).Scan(&locks).Error
	if err != nil || len(locks) == 0 {
		return nil, err
	}
	held := make(map[[2]int64]struct{}, len(locks))
	for _, lock := range locks {
		held[[2]int64{lock.ClassID, lock.ObjID}] = struct{}{}
	}
	var ids []uuid.UUID
	if err := tx.Model(&FactoryWorkOrderQueueItem{}).
		Where("organization_id = ?", orgID).
		Distinct("work_order_id").
		Pluck("work_order_id", &ids).Error; err != nil {
		return nil, err
	}
	var reserved []uuid.UUID
	for _, id := range ids {
		key := WorkOrderMergeAdvisoryKey(id)
		classID := int64(uint32(key >> 32))
		objID := int64(uint32(key))
		if _, ok := held[[2]int64{classID, objID}]; ok {
			reserved = append(reserved, id)
		}
	}
	return reserved, nil
}

func organizationAdmissionKey(orgID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("organization-task-admission:"), orgID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
