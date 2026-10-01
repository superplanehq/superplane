package models

import (
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DatadogWebhookOutcomeAccepted       = "accepted"
	DatadogWebhookOutcomeNoSubscription = "no_subscription"
	DatadogWebhookOutcomeIgnored        = "ignored"
	DatadogWebhookOutcomeRejected       = "rejected"
	DatadogWebhookOutcomeFailed         = "failed"
	DatadogWebhookOutcomePending        = "pending"

	// DatadogWebhookReceiptRetention is how long Installation Admin keeps a receipt.
	DatadogWebhookReceiptRetention = 14 * 24 * time.Hour
)

// DatadogWebhookReceipt is one incoming Datadog webhook. It stores the event
// type and the result. It does not store the payload, signature, or tokens.
type DatadogWebhookReceipt struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	ReceivedAt        time.Time
	IntegrationID     uuid.UUID
	OrganizationID    uuid.UUID
	EventType         string
	AlertTransition   string
	AlertID           string
	Service           string
	IssueID           string
	HTTPStatus        int
	Outcome           string
	SubscriptionCount int
	// TaskIDs lists SuperPlane tasks this webhook created. IDs are comma-separated.
	TaskIDs string
}

func (DatadogWebhookReceipt) TableName() string {
	return "datadog_webhook_receipts"
}

func (r DatadogWebhookReceipt) TaskIDList() []string {
	return splitCommaIDs(r.TaskIDs)
}

func CreateDatadogWebhookReceipt(tx *gorm.DB, receipt DatadogWebhookReceipt) (uuid.UUID, error) {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	if receipt.ReceivedAt.IsZero() {
		receipt.ReceivedAt = time.Now().UTC()
	}
	receipt.EventType = clipWebhookField(receipt.EventType)
	receipt.AlertTransition = clipWebhookField(receipt.AlertTransition)
	receipt.AlertID = clipWebhookField(receipt.AlertID)
	receipt.Service = clipWebhookField(receipt.Service)
	receipt.IssueID = clipWebhookField(receipt.IssueID)
	receipt.Outcome = clipWebhookField(receipt.Outcome)

	if err := tx.Create(&receipt).Error; err != nil {
		return uuid.Nil, err
	}
	return receipt.ID, nil
}

// DeleteExpiredDatadogWebhookReceipts removes receipts older than olderThan.
// It deletes at most limit rows so one pass cannot scan the whole table.
func DeleteExpiredDatadogWebhookReceipts(tx *gorm.DB, olderThan time.Time, limit int) (int64, error) {
	return deleteRowsLimited(tx, &DatadogWebhookReceipt{}, limit, "received_at < ?", olderThan)
}

func UpdateDatadogWebhookReceiptResult(tx *gorm.DB, id uuid.UUID, status int, outcome string, subscriptionCount int) error {
	if id == uuid.Nil {
		return nil
	}
	return tx.Model(&DatadogWebhookReceipt{}).Where("id = ?", id).Updates(map[string]any{
		"http_status":        status,
		"outcome":            clipWebhookField(outcome),
		"subscription_count": subscriptionCount,
	}).Error
}

// AppendDatadogWebhookTask records a SuperPlane task created from this webhook.
// A second append of the same task ID does not duplicate it.
func AppendDatadogWebhookTask(tx *gorm.DB, receiptID, taskID uuid.UUID) error {
	if tx == nil || receiptID == uuid.Nil || taskID == uuid.Nil {
		return nil
	}

	var receipt DatadogWebhookReceipt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", receiptID).First(&receipt).Error
	if err != nil {
		return err
	}

	ids := receipt.TaskIDList()
	if slices.Contains(ids, taskID.String()) {
		return nil
	}
	ids = append(ids, taskID.String())
	return tx.Model(&DatadogWebhookReceipt{}).Where("id = ?", receiptID).Update("task_ids", strings.Join(ids, ",")).Error
}

// CreateRejectedDatadogWebhookReceiptIfAllowed stores one rejected receipt when
// this integration is under the limit. A caller that is already at the limit
// stores nothing. oldest is the earliest rejected receipt in the window when
// the limit is already met, so the caller can skip later calls until it expires.
// The limit is counted before any lock. The lock is taken only when a receipt
// may still be stored.
func CreateRejectedDatadogWebhookReceiptIfAllowed(
	tx *gorm.DB,
	receipt DatadogWebhookReceipt,
	since time.Time,
	limit int,
) (uuid.UUID, bool, time.Time, error) {
	if tx == nil || limit <= 0 {
		return uuid.Nil, false, time.Time{}, nil
	}

	count, oldest, err := countRejectedDatadogWebhookReceipts(tx, receipt.IntegrationID, since)
	if err != nil {
		return uuid.Nil, false, time.Time{}, err
	}
	if count >= int64(limit) {
		return uuid.Nil, false, oldest, nil
	}

	var id uuid.UUID
	stored := false
	err = tx.Transaction(func(inner *gorm.DB) error {
		if err := lockRejectedDatadogWebhookReceipts(inner, receipt.IntegrationID); err != nil {
			return err
		}

		count, oldest, err = countRejectedDatadogWebhookReceipts(inner, receipt.IntegrationID, since)
		if err != nil || count >= int64(limit) {
			return err
		}

		receipt.Outcome = DatadogWebhookOutcomeRejected
		id, err = CreateDatadogWebhookReceipt(inner, receipt)
		if err != nil {
			return err
		}
		stored = true
		return nil
	})
	if err != nil {
		return uuid.Nil, false, time.Time{}, err
	}
	if !stored {
		return uuid.Nil, false, oldest, nil
	}
	return id, true, time.Time{}, nil
}

func UpdateDatadogWebhookReceiptSummary(tx *gorm.DB, id uuid.UUID, summary DatadogWebhookReceipt) error {
	if tx == nil || id == uuid.Nil {
		return nil
	}
	return tx.Model(&DatadogWebhookReceipt{}).Where("id = ?", id).Updates(map[string]any{
		"event_type":       clipWebhookField(summary.EventType),
		"alert_transition": clipWebhookField(summary.AlertTransition),
		"alert_id":         clipWebhookField(summary.AlertID),
		"service":          clipWebhookField(summary.Service),
		"issue_id":         clipWebhookField(summary.IssueID),
	}).Error
}

func ListDatadogWebhookReceipts(tx *gorm.DB, limit, offset int) ([]DatadogWebhookReceipt, int64, error) {
	var total int64
	if err := tx.Model(&DatadogWebhookReceipt{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var receipts []DatadogWebhookReceipt
	err := tx.Order("received_at DESC").Limit(limit).Offset(offset).Find(&receipts).Error
	return receipts, total, err
}

func countRejectedDatadogWebhookReceipts(tx *gorm.DB, integrationID uuid.UUID, since time.Time) (int64, time.Time, error) {
	if integrationID == uuid.Nil {
		return 0, time.Time{}, nil
	}

	var row struct {
		Count  int64
		Oldest *time.Time
	}
	err := tx.Model(&DatadogWebhookReceipt{}).
		Select("COUNT(*) AS count, MIN(received_at) AS oldest").
		Where("integration_id = ? AND outcome = ? AND received_at >= ?", integrationID, DatadogWebhookOutcomeRejected, since).
		Scan(&row).Error
	if err != nil {
		return 0, time.Time{}, err
	}
	if row.Oldest == nil {
		return row.Count, time.Time{}, nil
	}
	return row.Count, row.Oldest.UTC(), nil
}

// RejectedDatadogWebhookReceiptLockKey identifies the advisory lock that
// guards rejected receipt storage for one integration.
func RejectedDatadogWebhookReceiptLockKey(integrationID uuid.UUID) int64 {
	sum := sha256.Sum256(append([]byte("datadog-webhook-rejected:"), integrationID[:]...))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

func lockRejectedDatadogWebhookReceipts(tx *gorm.DB, integrationID uuid.UUID) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", RejectedDatadogWebhookReceiptLockKey(integrationID)).Error
}

func splitCommaIDs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
