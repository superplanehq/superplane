package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OrganizationSuspendedMessage           = "This organization is suspended. Existing data remains available. Configuration changes and new runs are blocked. Contact an installation admin."
	OrganizationSuspensionActionSuspend    = "suspend"
	OrganizationSuspensionActionReactivate = "reactivate"
)

var (
	ErrOrganizationSuspended         = errors.New("organization is suspended")
	ErrSuspensionReasonRequired      = errors.New("suspension reason must contain 1 to 2000 characters")
	ErrSuspensionCancellationPending = errors.New("suspension cancellation pending")
)

// RequireActiveOrganization coordinates admission with suspension. Call it in
// the admission transaction, before taking any run, execution, runner or task locks.
func RequireActiveOrganization(tx *gorm.DB, organizationID uuid.UUID) error {
	organization, err := LockOrganization(tx, organizationID)
	if err != nil {
		return err
	}
	if organization.IsSuspended() {
		return ErrOrganizationSuspended
	}
	return nil
}

func RequireActiveCanvasOrganization(tx *gorm.DB, canvasID uuid.UUID) error {
	canvas, err := FindUnscopedCanvasInTransaction(tx, canvasID)
	if err != nil {
		return err
	}
	return RequireActiveOrganization(tx, canvas.OrganizationID)
}

// OrganizationSuspendedAt includes completed intervals, so a delayed timer
// cannot replay an occurrence after reactivation.
func OrganizationSuspendedAt(tx *gorm.DB, organizationID uuid.UUID, at time.Time) (bool, error) {
	var suspended bool
	err := tx.Raw(`SELECT EXISTS (
		SELECT 1 FROM organization_suspension_events s
		WHERE s.organization_id = ? AND s.action = 'suspend' AND s.created_at <= ?
		AND NOT EXISTS (SELECT 1 FROM organization_suspension_events r
			WHERE r.organization_id = s.organization_id AND r.action = 'reactivate'
			AND r.created_at >= s.created_at AND r.created_at <= ?))`, organizationID, at, at).Scan(&suspended).Error
	return suspended, err
}

type OrganizationSuspensionEvent struct {
	ID                      uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID          uuid.UUID
	AccountID               uuid.UUID
	Action                  string
	Reason                  string
	CreatedAt               time.Time
	CancellationCompletedAt *time.Time
}

func (o *Organization) IsSuspended() bool {
	return o.SuspendedAt != nil
}

// Suspend records state and history in one short transaction. It also accepts
// a caller transaction; the receiver then reflects that uncommitted state.
// Lock the organization before any run, task, or cancellation event locks in
// future admission and cancellation code. These transitions lock only the organization.
func (o *Organization) Suspend(tx *gorm.DB, adminID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if length := utf8.RuneCountInString(reason); length < 1 || length > 2000 {
		return ErrSuspensionReasonRequired
	}
	var current *Organization
	err := tx.Transaction(func(inner *gorm.DB) error {
		locked, err := LockOrganization(inner, o.ID)
		if err != nil {
			return err
		}
		current = locked
		if locked.IsSuspended() {
			return nil
		}
		// PostgreSQL stores microsecond precision; keep the receiver and event aligned.
		now := time.Now().UTC().Truncate(time.Microsecond)
		if err := inner.Model(locked).Updates(map[string]any{
			"suspended_at": now, "suspended_by_account_id": adminID, "suspension_reason": reason,
		}).Error; err != nil {
			return err
		}
		return inner.Create(&OrganizationSuspensionEvent{
			OrganizationID: o.ID, AccountID: adminID,
			Action: OrganizationSuspensionActionSuspend, Reason: reason, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return err
	}
	*o = *current
	return nil
}

// Reactivate uses the same organization-first lock order as Suspend. Pending
// cancellation must finish before current state can return to active.
// With a caller transaction, the receiver reflects uncommitted state.
func (o *Organization) Reactivate(tx *gorm.DB, adminID uuid.UUID) error {
	var current *Organization
	err := tx.Transaction(func(inner *gorm.DB) error {
		locked, err := LockOrganization(inner, o.ID)
		if err != nil {
			return err
		}
		current = locked
		if !locked.IsSuspended() {
			return nil
		}
		var pending int64
		if err := inner.Model(&OrganizationSuspensionEvent{}).
			Where("organization_id = ? AND action = ? AND cancellation_completed_at IS NULL", o.ID, OrganizationSuspensionActionSuspend).
			Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return ErrSuspensionCancellationPending
		}
		if err := inner.Model(locked).Updates(map[string]any{
			"suspended_at": nil, "suspended_by_account_id": nil, "suspension_reason": "",
		}).Error; err != nil {
			return err
		}
		return inner.Create(&OrganizationSuspensionEvent{
			OrganizationID: o.ID, AccountID: adminID,
			Action: OrganizationSuspensionActionReactivate, CreatedAt: time.Now(),
		}).Error
	})
	if err != nil {
		return err
	}
	*o = *current
	return nil
}

// FindOrganizationSuspensionStatus reads current status without a transition lock.
func FindOrganizationSuspensionStatus(tx *gorm.DB, organizationID uuid.UUID) (*Organization, error) {
	var organization Organization
	err := tx.Select("id", "suspended_at", "suspended_by_account_id", "suspension_reason").
		First(&organization, "id = ?", organizationID).Error
	if err != nil {
		return nil, err
	}
	return &organization, nil
}

// ListOrganizationSuspensionEvents returns the complete organization history,
// newest first. IDs break timestamp ties to keep the order deterministic.
func ListOrganizationSuspensionEvents(tx *gorm.DB, organizationID uuid.UUID) ([]OrganizationSuspensionEvent, error) {
	events := []OrganizationSuspensionEvent{}
	err := tx.Where("organization_id = ?", organizationID).
		Order("created_at DESC").Order("id DESC").Find(&events).Error
	return events, err
}
