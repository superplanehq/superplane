package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SentryWebhookSourceHostedApp   = "hosted_app"
	SentryWebhookSourceIntegration = "integration"

	// Inbox states. The receipt outcome uses the same values.
	SentryWebhookStateQueued     = "queued"
	SentryWebhookStateProcessing = "processing"
	SentryWebhookStateRetrying   = "retrying"
	SentryWebhookStateBlocked    = "blocked"
	SentryWebhookStateProcessed  = "processed"
	SentryWebhookStateFiltered   = "filtered"

	// Target states. Pending and blocked targets run again. Processed,
	// filtered, and removed targets are finished.
	SentryWebhookTargetPending   = "pending"
	SentryWebhookTargetBlocked   = "blocked"
	SentryWebhookTargetProcessed = "processed"
	SentryWebhookTargetFiltered  = "filtered"
	SentryWebhookTargetRemoved   = "removed"

	// SentryWebhookPriorityInstallation puts installation lifecycle events
	// before issue events, so a new connection gets its grant first.
	SentryWebhookPriorityInstallation = 10
	SentryWebhookPriorityDefault      = 0

	// SentryWebhookInboxRetention is how long finished inbox metadata stays.
	SentryWebhookInboxRetention = 14 * 24 * time.Hour

	sentryWebhookInstallationResource = "installation"
	sentryWebhookClaimLock            = "sentry-webhook-inbox-claim"
	sentryWebhookErrorMaxBytes        = 1000
)

// ErrSentryWebhookLeaseLost means another worker owns the inbox entry now.
// The caller must not commit work for that entry.
var ErrSentryWebhookLeaseLost = errors.New("sentry webhook lease is no longer held")

// SentryWebhookInbox is one verified Sentry webhook delivery. SuperPlane
// stores every incoming request as its own entry. The payload stays
// encrypted, and SuperPlane removes it when every target is finished.
type SentryWebhookInbox struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	ReceiptID         *uuid.UUID
	Source            string
	IntegrationID     *uuid.UUID
	InstallationUUID  string
	HookResource      string
	Action            string
	Priority          int
	PayloadCiphertext []byte
	State             string
	TargetsResolved   bool
	Attempts          int
	NextAttemptAt     time.Time
	LeaseOwner        *uuid.UUID
	LeaseExpiresAt    *time.Time
	LastError         string
	ReceivedAt        time.Time
	CompletedAt       *time.Time
	UpdatedAt         time.Time
}

// SentryWebhookDeliveryTarget is one trigger subscription that must receive
// an inbox entry. SuperPlane tracks each target separately, so a failed
// target does not repeat the work of a target that already finished.
type SentryWebhookDeliveryTarget struct {
	ID            uuid.UUID `gorm:"primaryKey"`
	InboxID       uuid.UUID
	IntegrationID uuid.UUID
	WorkflowID    uuid.UUID
	NodeID        string
	State         string
	LastError     string
	CompletedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SentryAppInstallationDeletion records that Sentry uninstalled the public
// app. A delayed installation.created retry checks it, so it cannot store a
// grant for an installation that no longer exists.
type SentryAppInstallationDeletion struct {
	InstallationUUID string `gorm:"primaryKey"`
	DeletedAt        time.Time
}

// SentryWebhookInboxStats describes unfinished work for metrics and alerts.
type SentryWebhookInboxStats struct {
	Unfinished int64
	Blocked    int64
	OldestAge  time.Duration
}

func NewSentryWebhookInbox(source, resource string, now time.Time) SentryWebhookInbox {
	priority := SentryWebhookPriorityDefault
	if strings.TrimSpace(resource) == sentryWebhookInstallationResource {
		priority = SentryWebhookPriorityInstallation
	}

	return SentryWebhookInbox{
		ID:            uuid.New(),
		Source:        source,
		HookResource:  clipWebhookField(resource),
		Priority:      priority,
		State:         SentryWebhookStateQueued,
		NextAttemptAt: now,
		ReceivedAt:    now,
		UpdatedAt:     now,
	}
}

func NewSentryWebhookDeliveryTarget(inboxID, integrationID, workflowID uuid.UUID, nodeID string, now time.Time) SentryWebhookDeliveryTarget {
	return SentryWebhookDeliveryTarget{
		ID:            uuid.New(),
		InboxID:       inboxID,
		IntegrationID: integrationID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		State:         SentryWebhookTargetPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (SentryWebhookInbox) TableName() string {
	return "sentry_webhook_inbox"
}

func (SentryWebhookDeliveryTarget) TableName() string {
	return "sentry_webhook_delivery_targets"
}

func (SentryAppInstallationDeletion) TableName() string {
	return "sentry_app_installation_deletions"
}

func (e *SentryWebhookInbox) IsInstallationEvent() bool {
	return e.HookResource == sentryWebhookInstallationResource
}

// Finished reports whether the target needs no more work.
func (t *SentryWebhookDeliveryTarget) Finished() bool {
	switch t.State {
	case SentryWebhookTargetProcessed, SentryWebhookTargetFiltered, SentryWebhookTargetRemoved:
		return true
	default:
		return false
	}
}

// CreateSentryWebhookDelivery stores a verified delivery: the receipt, the
// inbox entry, and the targets known at acceptance. The caller runs it in
// one transaction, and answers Sentry only after that transaction commits.
func CreateSentryWebhookDelivery(tx *gorm.DB, receipt SentryWebhookReceipt, entry *SentryWebhookInbox, targets []SentryWebhookDeliveryTarget) error {
	receiptID, err := CreateSentryWebhookReceipt(tx, receipt)
	if err != nil {
		return err
	}

	entry.ReceiptID = &receiptID
	entry.InstallationUUID = clipWebhookField(entry.InstallationUUID)
	entry.Action = clipWebhookField(entry.Action)
	entry.LastError = clipSentryWebhookError(entry.LastError)
	if err := tx.Create(entry).Error; err != nil {
		return err
	}

	if len(targets) == 0 {
		return nil
	}
	return tx.Create(&targets).Error
}

// ClaimSentryWebhookInbox leases up to limit due entries for one worker.
// A transaction-scoped advisory lock serializes claims, so the check for a
// leased installation event sees every committed lease. SKIP LOCKED keeps
// the claim from waiting for a completion transaction that holds a row.
// Two lifecycle events of one installation never run at the same time.
func ClaimSentryWebhookInbox(tx *gorm.DB, owner uuid.UUID, now time.Time, lease time.Duration, limit int) ([]SentryWebhookInbox, error) {
	if limit <= 0 {
		return nil, nil
	}

	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", sentryWebhookClaimLock).Error; err != nil {
		return nil, err
	}

	var candidates []SentryWebhookInbox
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("completed_at IS NULL").
		Where("next_attempt_at <= ?", now).
		Where("(lease_expires_at IS NULL OR lease_expires_at <= ?)", now).
		Where(`NOT (
			hook_resource = ? AND installation_uuid <> '' AND EXISTS (
				SELECT 1 FROM sentry_webhook_inbox AS other
				WHERE other.hook_resource = sentry_webhook_inbox.hook_resource
				  AND other.installation_uuid = sentry_webhook_inbox.installation_uuid
				  AND other.id <> sentry_webhook_inbox.id
				  AND other.completed_at IS NULL
				  AND other.lease_expires_at > ?
			)
		)`, sentryWebhookInstallationResource, now).
		Order("priority DESC, next_attempt_at ASC, received_at ASC").
		Limit(limit).
		Find(&candidates).
		Error
	if err != nil {
		return nil, err
	}

	claimed := oneLifecycleEventPerInstallation(candidates)
	if len(claimed) == 0 {
		return nil, nil
	}

	leaseExpiresAt := now.Add(lease)
	ids := make([]uuid.UUID, 0, len(claimed))
	receiptIDs := make([]uuid.UUID, 0, len(claimed))
	for i := range claimed {
		ids = append(ids, claimed[i].ID)
		if claimed[i].ReceiptID != nil {
			receiptIDs = append(receiptIDs, *claimed[i].ReceiptID)
		}
		claimed[i].State = SentryWebhookStateProcessing
		claimed[i].LeaseOwner = &owner
		claimed[i].LeaseExpiresAt = &leaseExpiresAt
		claimed[i].Attempts++
		claimed[i].UpdatedAt = now
	}

	err = tx.Model(&SentryWebhookInbox{}).Where("id IN ?", ids).Updates(map[string]any{
		"state":            SentryWebhookStateProcessing,
		"lease_owner":      owner,
		"lease_expires_at": leaseExpiresAt,
		"attempts":         gorm.Expr("attempts + 1"),
		"updated_at":       now,
	}).Error
	if err != nil {
		return nil, err
	}

	if err := updateSentryWebhookReceiptOutcomes(tx, receiptIDs, SentryWebhookStateProcessing); err != nil {
		return nil, err
	}
	return claimed, nil
}

// RenewLease extends the lease when this worker still owns it. It returns
// false when the lease is lost, so the caller can cancel its work.
func (e *SentryWebhookInbox) RenewLease(tx *gorm.DB, now time.Time, lease time.Duration) (bool, error) {
	if e.LeaseOwner == nil {
		return false, nil
	}

	leaseExpiresAt := now.Add(lease)
	result := tx.Model(&SentryWebhookInbox{}).
		Where("id = ? AND lease_owner = ? AND lease_expires_at > ? AND completed_at IS NULL", e.ID, *e.LeaseOwner, now).
		Updates(map[string]any{"lease_expires_at": leaseExpiresAt, "updated_at": now})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}

	e.LeaseExpiresAt = &leaseExpiresAt
	return true, nil
}

// LockLease locks the entry row and confirms that this worker still owns an
// unexpired lease. Every completion transaction calls it first, so a worker
// whose lease expired cannot commit events or state.
func (e *SentryWebhookInbox) LockLease(tx *gorm.DB, now time.Time) error {
	if e.LeaseOwner == nil {
		return ErrSentryWebhookLeaseLost
	}

	var locked SentryWebhookInbox
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND lease_owner = ? AND lease_expires_at > ? AND completed_at IS NULL", e.ID, *e.LeaseOwner, now).
		First(&locked).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrSentryWebhookLeaseLost
	}
	return err
}

func (e *SentryWebhookInbox) ListTargets(tx *gorm.DB) ([]SentryWebhookDeliveryTarget, error) {
	var targets []SentryWebhookDeliveryTarget
	err := tx.Where("inbox_id = ?", e.ID).Order("created_at ASC, id ASC").Find(&targets).Error
	return targets, err
}

// ResolveTargets stores the targets that SuperPlane found after acceptance,
// for example after a missing connection is repaired. The caller holds the
// lease in the same transaction.
func (e *SentryWebhookInbox) ResolveTargets(tx *gorm.DB, targets []SentryWebhookDeliveryTarget, now time.Time) error {
	if len(targets) > 0 {
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "inbox_id"}, {Name: "integration_id"}, {Name: "workflow_id"}, {Name: "node_id"}},
			DoNothing: true,
		}).Create(&targets).Error
		if err != nil {
			return err
		}
	}

	e.TargetsResolved = true
	e.UpdatedAt = now
	return tx.Model(&SentryWebhookInbox{}).Where("id = ?", e.ID).Updates(map[string]any{
		"targets_resolved": true,
		"updated_at":       now,
	}).Error
}

// Finish records the final state of a target.
func (t *SentryWebhookDeliveryTarget) Finish(tx *gorm.DB, state, note string, now time.Time) error {
	t.State = state
	t.LastError = clipSentryWebhookError(note)
	t.CompletedAt = &now
	t.UpdatedAt = now
	return tx.Model(&SentryWebhookDeliveryTarget{}).Where("id = ?", t.ID).Updates(map[string]any{
		"state":        t.State,
		"last_error":   t.LastError,
		"completed_at": now,
		"updated_at":   now,
	}).Error
}

// Defer records why a target must run again. State is pending for a
// temporary failure and blocked when an operator or user must act.
func (t *SentryWebhookDeliveryTarget) Defer(tx *gorm.DB, state, reason string, now time.Time) error {
	t.State = state
	t.LastError = clipSentryWebhookError(reason)
	t.UpdatedAt = now
	return tx.Model(&SentryWebhookDeliveryTarget{}).
		Where("id = ? AND completed_at IS NULL", t.ID).
		Updates(map[string]any{
			"state":      t.State,
			"last_error": t.LastError,
			"updated_at": now,
		}).Error
}

// Complete finishes the entry. It drops the encrypted payload, keeps the
// metadata for the retention period, and updates the receipt outcome.
func (e *SentryWebhookInbox) Complete(tx *gorm.DB, state, note string, now time.Time) error {
	e.State = state
	e.LastError = clipSentryWebhookError(note)
	e.PayloadCiphertext = nil
	e.CompletedAt = &now
	e.LeaseOwner = nil
	e.LeaseExpiresAt = nil
	e.UpdatedAt = now

	err := tx.Model(&SentryWebhookInbox{}).Where("id = ?", e.ID).Updates(map[string]any{
		"state":              e.State,
		"last_error":         e.LastError,
		"payload_ciphertext": nil,
		"completed_at":       now,
		"lease_owner":        nil,
		"lease_expires_at":   nil,
		"updated_at":         now,
	}).Error
	if err != nil {
		return err
	}
	return e.updateReceipt(tx)
}

// Reschedule releases the lease and plans the next attempt. It never
// deletes the entry, however many attempts failed.
func (e *SentryWebhookInbox) Reschedule(tx *gorm.DB, state, reason string, nextAttemptAt, now time.Time) error {
	e.State = state
	e.LastError = clipSentryWebhookError(reason)
	e.NextAttemptAt = nextAttemptAt
	e.LeaseOwner = nil
	e.LeaseExpiresAt = nil
	e.UpdatedAt = now

	err := tx.Model(&SentryWebhookInbox{}).Where("id = ?", e.ID).Updates(map[string]any{
		"state":            e.State,
		"last_error":       e.LastError,
		"next_attempt_at":  nextAttemptAt,
		"lease_owner":      nil,
		"lease_expires_at": nil,
		"updated_at":       now,
	}).Error
	if err != nil {
		return err
	}
	return e.updateReceipt(tx)
}

func (e *SentryWebhookInbox) updateReceipt(tx *gorm.DB) error {
	if e.ReceiptID == nil {
		return nil
	}
	return tx.Model(&SentryWebhookReceipt{}).Where("id = ?", *e.ReceiptID).Updates(map[string]any{
		"outcome":          e.State,
		"processing_error": e.LastError,
	}).Error
}

// DeleteExpiredSentryWebhookInbox removes finished entries older than
// olderThan, and their targets. Unfinished entries are never selected.
func DeleteExpiredSentryWebhookInbox(tx *gorm.DB, olderThan time.Time, limit int) (int64, error) {
	return deleteRowsLimited(tx, &SentryWebhookInbox{}, limit, "completed_at IS NOT NULL AND completed_at < ?", olderThan)
}

// FindSentryWebhookInboxStats counts unfinished entries for metrics.
func FindSentryWebhookInboxStats(tx *gorm.DB, now time.Time) (SentryWebhookInboxStats, error) {
	var row struct {
		Unfinished int64
		Blocked    int64
		Oldest     *time.Time
	}
	err := tx.Model(&SentryWebhookInbox{}).
		Select("COUNT(*) AS unfinished, COUNT(*) FILTER (WHERE state = ?) AS blocked, MIN(received_at) AS oldest", SentryWebhookStateBlocked).
		Where("completed_at IS NULL").
		Scan(&row).
		Error
	if err != nil {
		return SentryWebhookInboxStats{}, err
	}

	stats := SentryWebhookInboxStats{Unfinished: row.Unfinished, Blocked: row.Blocked}
	if row.Oldest != nil {
		stats.OldestAge = now.Sub(*row.Oldest)
	}
	return stats, nil
}

// RecordSentryAppInstallationDeletion stores that Sentry deleted an
// installation. It is idempotent.
func RecordSentryAppInstallationDeletion(tx *gorm.DB, installationUUID string, deletedAt time.Time) error {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&SentryAppInstallationDeletion{
		InstallationUUID: installationUUID,
		DeletedAt:        deletedAt,
	}).Error
}

func SentryAppInstallationDeleted(tx *gorm.DB, installationUUID string) (bool, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	if installationUUID == "" {
		return false, nil
	}
	var count int64
	err := tx.Model(&SentryAppInstallationDeletion{}).Where("installation_uuid = ?", installationUUID).Count(&count).Error
	return count > 0, err
}

// LockSentryAppInstallation serializes grant writes and deletion records for
// one installation until the transaction ends.
func LockSentryAppInstallation(tx *gorm.DB, installationUUID string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "sentry-installation:"+strings.TrimSpace(installationUUID)).Error
}

func ReadSentryWebhookInboxPayloadPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("sentry_webhook_inbox").
		Select("id::text, payload_ciphertext, id::text, ''").
		Where("payload_ciphertext IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceSentryWebhookInboxPayload(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE sentry_webhook_inbox SET payload_ciphertext = ? WHERE id = CAST(? AS uuid) AND payload_ciphertext = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func oneLifecycleEventPerInstallation(candidates []SentryWebhookInbox) []SentryWebhookInbox {
	seen := map[string]bool{}
	claimed := make([]SentryWebhookInbox, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.IsInstallationEvent() && candidate.InstallationUUID != "" {
			if seen[candidate.InstallationUUID] {
				continue
			}
			seen[candidate.InstallationUUID] = true
		}
		claimed = append(claimed, candidate)
	}
	return claimed
}

func updateSentryWebhookReceiptOutcomes(tx *gorm.DB, receiptIDs []uuid.UUID, outcome string) error {
	if len(receiptIDs) == 0 {
		return nil
	}
	return tx.Model(&SentryWebhookReceipt{}).Where("id IN ?", receiptIDs).Update("outcome", outcome).Error
}

func clipSentryWebhookError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= sentryWebhookErrorMaxBytes {
		return value
	}
	end := sentryWebhookErrorMaxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}
