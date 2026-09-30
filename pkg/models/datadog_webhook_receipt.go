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
// stores nothing. The check and the insert share one transaction.
func CreateRejectedDatadogWebhookReceiptIfAllowed(
	tx *gorm.DB,
	receipt DatadogWebhookReceipt,
	since time.Time,
	limit int,
) (uuid.UUID, bool, error) {
	if tx == nil {
		return uuid.Nil, false, nil
	}

	var id uuid.UUID
	stored := false
	err := tx.Transaction(func(inner *gorm.DB) error {
		allowed, err := rejectedDatadogWebhookReceiptAllowed(inner, receipt.IntegrationID, since, limit)
		if err != nil || !allowed {
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
		return uuid.Nil, false, err
	}
	return id, stored, nil
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

func rejectedDatadogWebhookReceiptAllowed(tx *gorm.DB, integrationID uuid.UUID, since time.Time, limit int) (bool, error) {
	if integrationID == uuid.Nil || limit <= 0 {
		return false, nil
	}
	if err := lockRejectedDatadogWebhookReceipts(tx, integrationID); err != nil {
		return false, err
	}

	var count int64
	err := tx.Model(&DatadogWebhookReceipt{}).
		Where("integration_id = ? AND outcome = ? AND received_at >= ?", integrationID, DatadogWebhookOutcomeRejected, since).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count < int64(limit), nil
}

func lockRejectedDatadogWebhookReceipts(tx *gorm.DB, integrationID uuid.UUID) error {
	sum := sha256.Sum256(append([]byte("datadog-webhook-rejected:"), integrationID[:]...))
	key := int64(binary.BigEndian.Uint64(sum[:8]))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", key).Error
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
