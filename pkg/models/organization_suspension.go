package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const OrganizationSuspendedMessage = "This organization is suspended. You can view existing data, but cannot start new work."
const SuspensionReasonRequiredMessage = "Enter a suspension reason between 1 and 2000 characters."
const SuspensionCancellationPendingMessage = "Work is still stopping. Reactivate the organization after cancellation completes."

var (
	ErrOrganizationSuspended         = errors.New("organization suspended")
	ErrSuspensionReasonRequired      = errors.New("suspension reason required")
	ErrSuspensionCancellationPending = errors.New("suspension cancellation pending")
)

type OrganizationSuspensionEvent struct {
	ID                      uuid.UUID  `gorm:"primaryKey;default:uuid_generate_v4()" json:"id"`
	OrganizationID          uuid.UUID  `json:"organization_id"`
	AccountID               uuid.UUID  `json:"account_id"`
	Action                  string     `json:"action"`
	Reason                  string     `json:"reason"`
	CreatedAt               time.Time  `json:"created_at"`
	CancellationCompletedAt *time.Time `json:"cancellation_completed_at"`
}

func (o *Organization) IsSuspended() bool { return o.SuspendedAt != nil }

func RequireOrganizationActive(tx *gorm.DB, id uuid.UUID) error {
	var organization Organization
	if err := tx.Select("id", "suspended_at").First(&organization, "id = ?", id).Error; err != nil {
		return err
	}
	if organization.IsSuspended() {
		return ErrOrganizationSuspended
	}
	return nil
}

func RequireCanvasOrganizationActive(tx *gorm.DB, canvasID uuid.UUID) error {
	var organization Organization
	err := tx.Model(&Organization{}).
		Select("organizations.id", "organizations.suspended_at").
		Joins("JOIN workflows ON workflows.organization_id = organizations.id").
		Where("workflows.id = ? AND workflows.deleted_at IS NULL", canvasID).First(&organization).Error
	if err != nil {
		return err
	}
	if organization.IsSuspended() {
		return ErrOrganizationSuspended
	}
	return nil
}

func (o *Organization) Suspend(tx *gorm.DB, adminID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > 2000 {
		return ErrSuspensionReasonRequired
	}
	return tx.Transaction(func(tx *gorm.DB) error {
		locked, err := LockOrganization(tx, o.ID)
		if err != nil {
			return err
		}
		if locked.IsSuspended() {
			*o = *locked
			return nil
		}
		now := time.Now()
		if err := tx.Model(locked).Updates(map[string]any{"suspended_at": now, "suspended_by_account_id": adminID, "suspension_reason": reason}).Error; err != nil {
			return err
		}
		if err := tx.Create(&OrganizationSuspensionEvent{OrganizationID: o.ID, AccountID: adminID, Action: "suspend", Reason: reason}).Error; err != nil {
			return err
		}
		*o = *locked
		return nil
	})
}

func (o *Organization) Reactivate(tx *gorm.DB, adminID uuid.UUID) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		locked, err := LockOrganization(tx, o.ID)
		if err != nil {
			return err
		}
		if !locked.IsSuspended() {
			*o = *locked
			return nil
		}
		var pending int64
		if err := tx.Model(&OrganizationSuspensionEvent{}).Where("organization_id = ? AND action = 'suspend' AND cancellation_completed_at IS NULL", o.ID).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return ErrSuspensionCancellationPending
		}
		if err := tx.Model(locked).Updates(map[string]any{"suspended_at": nil, "suspended_by_account_id": nil, "suspension_reason": ""}).Error; err != nil {
			return err
		}
		if err := tx.Create(&OrganizationSuspensionEvent{OrganizationID: o.ID, AccountID: adminID, Action: "reactivate"}).Error; err != nil {
			return err
		}
		*o = *locked
		return nil
	})
}

func ListOrganizationSuspensionEvents(tx *gorm.DB, organizationID uuid.UUID) ([]OrganizationSuspensionEvent, error) {
	events := []OrganizationSuspensionEvent{}
	err := tx.Where("organization_id = ?", organizationID).Order("created_at DESC").Limit(100).Find(&events).Error
	return events, err
}

func ListPendingOrganizationSuspensions(tx *gorm.DB) ([]OrganizationSuspensionEvent, error) {
	var events []OrganizationSuspensionEvent
	err := tx.Where("action = 'suspend' AND cancellation_completed_at IS NULL").Order("created_at").Limit(100).Find(&events).Error
	return events, err
}

func LockPendingOrganizationSuspension(tx *gorm.DB, id uuid.UUID) (*OrganizationSuspensionEvent, error) {
	var event OrganizationSuspensionEvent
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ? AND action = 'suspend' AND cancellation_completed_at IS NULL", id).First(&event).Error
	return &event, err
}

// SuspensionWork returns bounded batches. Cancelling runs continue through the
// normal finalizer; runner completion and log uploads remain available.
func (o *Organization) SuspensionWork(tx *gorm.DB) ([]CanvasRun, []RunnerTask, []AgentSession, error) {
	var runs []CanvasRun
	err := tx.Model(&CanvasRun{}).Joins("JOIN workflows ON workflows.id = workflow_runs.workflow_id").
		Where("workflows.organization_id = ? AND workflow_runs.state IN ?", o.ID, []string{CanvasRunStatePending, CanvasRunStateStarted}).
		Order("workflow_runs.created_at").Limit(100).Find(&runs).Error
	if err != nil {
		return nil, nil, nil, err
	}
	var tasks []RunnerTask
	err = tx.Where("organization_id = ? AND state IN ?", o.ID, []string{RunnerTaskStateQueued, RunnerTaskStateReserved, RunnerTaskStateRunning}).Order("queued_at").Limit(100).Find(&tasks).Error
	if err != nil {
		return nil, nil, nil, err
	}
	var sessions []AgentSession
	err = tx.Where("organization_id = ? AND status = ?", o.ID, AgentSessionStatusStreaming).Limit(100).Find(&sessions).Error
	return runs, tasks, sessions, err
}

func (o *Organization) CompleteSuspensionCancellation(tx *gorm.DB, eventID uuid.UUID) error {
	var queued int64
	if err := tx.Model(&FactoryWorkOrderQueueItem{}).Where("organization_id = ?", o.ID).Count(&queued).Error; err != nil {
		return err
	}
	if queued > 0 {
		return nil
	}
	if err := o.pendingSuspendedRequests(tx).Count(&queued).Error; err != nil {
		return err
	}
	if queued > 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&CanvasRun{}).Joins("JOIN workflows ON workflows.id = workflow_runs.workflow_id").
		Where("workflows.organization_id = ? AND workflow_runs.state <> ?", o.ID, CanvasRunStateFinished).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	runs, tasks, sessions, err := o.SuspensionWork(tx)
	if err != nil {
		return err
	}
	if len(runs)+len(tasks)+len(sessions) > 0 {
		return nil
	}
	return tx.Model(&OrganizationSuspensionEvent{}).Where("id = ? AND cancellation_completed_at IS NULL", eventID).Update("cancellation_completed_at", time.Now()).Error
}

// DiscardSuspendedWork removes queued activity without changing workflow
// configuration. Schedule hooks are retained so they can advance future ticks.
func (o *Organization) DiscardSuspendedWork(tx *gorm.DB) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		var items []FactoryWorkOrderQueueItem
		if err := tx.Where("organization_id = ?", o.ID).Limit(100).Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			dispatch, err := FindWorkOrderLineDispatch(tx, item.LineDispatchID)
			if err != nil {
				return err
			}
			if err := dispatch.Finish(tx, CanvasRunResultCancelled); err != nil {
				return err
			}
			if err := item.Delete(tx); err != nil {
				return err
			}
		}
		var ids []uuid.UUID
		if err := o.pendingSuspendedRequests(tx).
			Order("created_at").Limit(100).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		return tx.Model(&CanvasNodeRequest{}).Where("id IN ?", ids).
			Updates(map[string]any{"state": NodeExecutionRequestStateCompleted, "updated_at": time.Now()}).Error
	})
}

func (o *Organization) pendingSuspendedRequests(tx *gorm.DB) *gorm.DB {
	return tx.Model(&CanvasNodeRequest{}).
		Where("state = ?", NodeExecutionRequestStatePending).
		Where("workflow_id IN (?)", tx.Model(&Canvas{}).Select("id").Where("organization_id = ?", o.ID)).
		Where(`NOT EXISTS (SELECT 1 FROM workflow_nodes n WHERE n.workflow_id = workflow_node_requests.workflow_id
		 AND n.node_id = workflow_node_requests.node_id AND n.ref->'trigger'->>'name' = 'schedule'
		 AND workflow_node_requests.execution_id IS NULL)`)
}

// FilterOrganizationSuspension applies the installation-admin list filter.
func FilterOrganizationSuspension(tx *gorm.DB, status string) *gorm.DB {
	// The organization list builds aggregate subqueries from this DB handle.
	// Apply the filter only to organization queries and keep the handle reusable.
	return tx.Scopes(func(query *gorm.DB) *gorm.DB {
		if _, ok := query.Statement.Model.(*Organization); !ok {
			return query
		}
		switch status {
		case "suspended":
			return query.Where("organizations.suspended_at IS NOT NULL")
		case "active":
			return query.Where("organizations.suspended_at IS NULL")
		default:
			return query
		}
	}).Session(&gorm.Session{})
}
