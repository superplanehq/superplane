package models

import (
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSubscriptionModelUnavailable = newSubscriptionAdmissionError("the selected model is unavailable for this task owner; select another model or clear the override")
	ErrSubscriptionOwnerRequired    = newSubscriptionAdmissionError("Assign one eligible owner before you start this task")
	ErrSubscriptionOwnerInvalid     = newSubscriptionAdmissionError("Select an active member with a connected ChatGPT subscription in this workspace")
	ErrSubscriptionOwnerBusy        = errors.New("Stop this task and wait for its runners to stop before you change its owner")
	ErrSubscriptionTaskAssociation  = newSubscriptionAdmissionError("The run must belong to one task in this workspace")
)

func (o *FactoryWorkOrder) UsesCodexSubscription(tx *gorm.DB) (bool, error) {
	f, err := FindFactory(tx.Unscoped(), o.OrganizationID, o.FactoryID)
	if err != nil {
		return false, err
	}
	return f.OnboardingConfigValue().CodexSubscription, nil
}

func SubscriptionOwnerEligible(tx *gorm.DB, organizationID, factoryID, userID uuid.UUID) (bool, error) {
	if _, err := ActiveCodexMember(tx, organizationID, userID); err != nil {
		if errors.Is(err, ErrCodexConnectionNotEligible) || errors.Is(err, ErrCodexConnectionUnavailable) {
			return false, nil
		}
		return false, err
	}
	c, err := FindFactoryCodexConnection(tx, organizationID, factoryID, userID)
	if errors.Is(err, ErrCodexConnectionNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return c.Status == CodexConnectionStatusReady && c.RevocationPendingAt == nil && c.CredentialVersion > 0 && len(c.BundleCiphertext) > 0, nil
}

// RequireSubscriptionOwner serializes admission with assignment. Workers that
// already hold execution locks skip a busy task and retry after their commit.
func (o *FactoryWorkOrder) RequireSubscriptionOwner(tx *gorm.DB, nonblocking bool) (uuid.UUID, error) {
	locking := clause.Locking{Strength: "UPDATE"}
	if nonblocking {
		locking.Options = "SKIP LOCKED"
	}
	var locked FactoryWorkOrder
	err := tx.Clauses(locking).Where("id = ? AND factory_id = ? AND organization_id = ?", o.ID, o.FactoryID, o.OrganizationID).First(&locked).Error
	if nonblocking && errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, &CodexConnectionBusyError{}
	}
	if err != nil {
		return uuid.Nil, err
	}
	owners, err := locked.ListAssignees(tx)
	if err != nil {
		return uuid.Nil, err
	}
	if len(owners) != 1 {
		return uuid.Nil, ErrSubscriptionOwnerRequired
	}
	eligible, err := SubscriptionOwnerEligible(tx, o.OrganizationID, o.FactoryID, owners[0].UserID)
	if err != nil {
		return uuid.Nil, err
	}
	if !eligible {
		if _, err := ActiveCodexMember(tx, o.OrganizationID, owners[0].UserID); err != nil {
			if errors.Is(err, ErrCodexConnectionNotEligible) || errors.Is(err, ErrCodexConnectionUnavailable) {
				return uuid.Nil, ErrSubscriptionOwnerInvalid
			}
			return uuid.Nil, err
		}
		c, lookupErr := FindFactoryCodexConnection(tx, o.OrganizationID, o.FactoryID, owners[0].UserID)
		if lookupErr != nil && !errors.Is(lookupErr, ErrCodexConnectionNotFound) {
			return uuid.Nil, lookupErr
		}
		if lookupErr == nil && c.RevocationPendingAt == nil && slices.Contains([]string{CodexConnectionStatusPending, CodexConnectionStatusFinishing}, c.Status) {
			login, err := FindCodexLoginLease(tx, c.ID)
			if err != nil {
				return uuid.Nil, err
			}
			if login != nil && login.RevokedAt == nil {
				return uuid.Nil, &CodexConnectionBusyError{TaskID: login.TaskID}
			}
		}
		return uuid.Nil, ErrSubscriptionOwnerInvalid
	}
	return owners[0].UserID, nil
}

func (o *FactoryWorkOrder) RequireExecutionOwner(tx *gorm.DB, nonblocking bool) error {
	subscription, err := o.UsesCodexSubscription(tx)
	if err != nil || !subscription {
		return err
	}
	_, err = o.RequireSubscriptionOwner(tx, nonblocking)
	return err
}

// SubscriptionOwnershipBlocked includes queued admissions, associated runs,
// and runners whose cancellation or provider termination is incomplete.
func (o *FactoryWorkOrder) SubscriptionOwnershipBlocked(tx *gorm.DB) (bool, error) {
	var queued int64
	if err := tx.Model(&FactoryWorkOrderQueueItem{}).Where("work_order_id = ?", o.ID).Count(&queued).Error; err != nil {
		return false, err
	}
	if queued > 0 {
		return true, nil
	}
	var executions int64
	if err := tx.Model(&FactoryWorkOrderExecution{}).Where("work_order_id = ? AND status IN ?", o.ID, []string{FactoryWorkOrderExecutionStatusPending, FactoryWorkOrderExecutionStatusRunning}).Count(&executions).Error; err != nil {
		return false, err
	}
	if executions > 0 {
		return true, nil
	}
	runs, err := o.ListRuns(tx)
	if err != nil {
		return false, err
	}
	for _, run := range runs {
		if run.State != CanvasRunStateFinished {
			return true, nil
		}
	}
	ids := make([]uuid.UUID, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}
	if len(ids) == 0 {
		return false, nil
	}
	var active int64
	err = tx.Table("runner_tasks t").Joins("JOIN workflow_node_execution_kvs kv ON kv.key = 'task_id' AND kv.value = t.id::text").
		Joins("JOIN workflow_node_executions e ON e.id = kv.execution_id AND e.workflow_id = kv.workflow_id AND e.node_id = kv.node_id").
		Joins("LEFT JOIN runners r ON r.id = t.runner_id").
		Where("e.run_id IN ?", ids).
		Where("t.state IN ? OR (r.id IS NOT NULL AND r.provider_deleted_at IS NULL AND (t.cancel_requested_at IS NOT NULL OR t.state = ?))", []string{RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}, RunnerTaskStateLost).Count(&active).Error
	return active > 0, err
}

// DefaultSubscriptionAssignees applies only to implicit assignments. Explicit
// assignments go through ReplaceAssignees and are rejected without mutation.
func (f *Factory) DefaultSubscriptionAssignees(tx *gorm.DB, userID *uuid.UUID) ([]uuid.UUID, error) {
	if userID == nil {
		return nil, nil
	}
	if !f.OnboardingConfigValue().CodexSubscription {
		return []uuid.UUID{*userID}, nil
	}
	eligible, err := SubscriptionOwnerEligible(tx.Clauses(clause.Locking{Strength: "SHARE"}).Session(&gorm.Session{}), f.OrganizationID, f.ID, *userID)
	if err != nil || !eligible {
		return nil, err
	}
	return []uuid.UUID{*userID}, nil
}

// FindSubscriptionWorkOrderForRun uses persisted associations and ancestors.
// Event payloads and agent-supplied task IDs are never authority for routing.
func FindSubscriptionWorkOrderForRun(tx *gorm.DB, orgID, factoryID, runID uuid.UUID) (*FactoryWorkOrder, error) {
	var orders []FactoryWorkOrder
	err := tx.Raw(factoryTaskRunAssociationsCTE+`
 SELECT DISTINCT o.* FROM factory_work_orders o
 JOIN associated_task_runs a ON a.work_order_id = o.id WHERE a.id = ?`, orgID, factoryID, runID).Scan(&orders).Error
	if err != nil {
		return nil, err
	}
	if len(orders) != 1 || orders[0].OrganizationID != orgID || orders[0].FactoryID != factoryID {
		return nil, ErrSubscriptionTaskAssociation
	}
	return &orders[0], nil
}

// ResolveSubscriptionCatalogOwner never substitutes another member's catalog.
func (f *Factory) ResolveSubscriptionCatalogOwner(tx *gorm.DB, callerID uuid.UUID, taskRef string) (uuid.UUID, error) {
	if _, err := ActiveCodexMember(tx, f.OrganizationID, callerID); err != nil {
		return uuid.Nil, err
	}
	if taskRef == "" {
		return callerID, nil
	}
	order, err := f.FindWorkOrderByRef(tx, taskRef)
	if err != nil {
		return uuid.Nil, err
	}
	owners, err := order.ListAssignees(tx)
	if err != nil {
		return uuid.Nil, err
	}
	if len(owners) != 1 {
		return uuid.Nil, nil
	}
	eligible, err := SubscriptionOwnerEligible(tx, f.OrganizationID, f.ID, owners[0].UserID)
	if err != nil || !eligible {
		return uuid.Nil, err
	}
	return owners[0].UserID, nil
}

// ValidateSubscriptionCanvasModels validates effective choices before admission.
// Historical dispatch snapshots remain unchanged when a catalog changes.
func (o *FactoryWorkOrder) ValidateSubscriptionCanvasModels(tx *gorm.DB, canvasID uuid.UUID, override string) error {
	subscription, err := o.UsesCodexSubscription(tx)
	if err != nil || !subscription {
		return err
	}
	ownerID, err := o.RequireSubscriptionOwner(tx, true)
	if err != nil {
		return err
	}
	version, err := FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return err
	}
	for _, node := range version.Nodes {
		credentials, _ := node.Configuration["credentials"].(map[string]any)
		if node.ComponentName() != "runnerCodex" || credentials["source"] != "subscription" {
			continue
		}
		model := strings.TrimSpace(override)
		if model == "" {
			model, _ = node.Configuration["model"].(string)
			model = strings.TrimSpace(model)
		}
		// Expressions resolve later. Dispatch checks the resolved value.
		if model == "" || strings.Contains(model, "{{") {
			continue
		}
		allowed, err := RunnerAcceptsModel(tx, o.OrganizationID, &o.FactoryID, node.ComponentName(), node.Configuration, model, ownerID)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrSubscriptionModelUnavailable
		}
	}
	return nil
}

func (o *FactoryWorkOrder) validateSubscriptionAssignment(tx *gorm.DB, ids []uuid.UUID) error {
	subscription, err := o.UsesCodexSubscription(tx)
	if err != nil || !subscription {
		return err
	}
	if err := o.LockForUpdate(tx); err != nil {
		return err
	}
	previous, err := o.ListAssignees(tx)
	if err != nil {
		return err
	}
	oldIDs := make([]uuid.UUID, 0, len(previous))
	for _, owner := range previous {
		oldIDs = append(oldIDs, owner.UserID)
	}
	if slices.Equal(oldIDs, ids) {
		return nil
	}
	if len(ids) > 1 {
		return ErrSubscriptionOwnerRequired
	}
	blocked, err := o.SubscriptionOwnershipBlocked(tx)
	if err != nil {
		return err
	}
	if blocked {
		return ErrSubscriptionOwnerBusy
	}
	if len(ids) == 0 {
		return nil
	}
	eligible, err := SubscriptionOwnerEligible(tx, o.OrganizationID, o.FactoryID, ids[0])
	if err != nil {
		return err
	}
	if !eligible {
		return ErrSubscriptionOwnerInvalid
	}
	return nil
}
