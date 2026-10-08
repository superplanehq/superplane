package models

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	LinearWebhookOutcomeAccepted       = "accepted"
	LinearWebhookOutcomeIgnored        = "ignored"
	LinearWebhookOutcomeNoSubscription = "no_subscription"
	LinearWebhookOutcomeRejected       = "rejected"
	LinearWebhookOutcomeFailed         = "failed"
	LinearWebhookOutcomePending        = "pending"

	// LinearWebhookReceiptRetention is how long Installation Admin keeps a receipt.
	LinearWebhookReceiptRetention = 14 * 24 * time.Hour
)

// LinearWebhookReceipt is one incoming Linear webhook. It stores the event
// type and the result. It does not store the payload, signature, or tokens.
type LinearWebhookReceipt struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	ReceivedAt        time.Time
	IntegrationID     uuid.UUID
	OrganizationID    uuid.UUID
	WebhookID         uuid.UUID
	EventType         string
	Action            string
	IssueIdentifier   string
	IssueID           string
	TeamKey           string
	WorkspaceKey      string
	HTTPStatus        int
	Outcome           string
	SubscriptionCount int
	// TaskIDs lists SuperPlane tasks this webhook created. IDs are comma-separated.
	TaskIDs string
	// DeliveryKey identifies the Linear payload. A later retry of the same
	// payload uses this key to skip a subscription that already accepted it.
	DeliveryKey string
}

func (LinearWebhookReceipt) TableName() string {
	return "linear_webhook_receipts"
}

func (r LinearWebhookReceipt) TaskIDList() []string {
	return splitCommaIDs(r.TaskIDs)
}

func CreateLinearWebhookReceipt(tx *gorm.DB, receipt LinearWebhookReceipt) (uuid.UUID, error) {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	if receipt.ReceivedAt.IsZero() {
		receipt.ReceivedAt = time.Now().UTC()
	}
	receipt.EventType = clipWebhookField(receipt.EventType)
	receipt.Action = clipWebhookField(receipt.Action)
	receipt.IssueIdentifier = clipWebhookField(receipt.IssueIdentifier)
	receipt.IssueID = clipWebhookField(receipt.IssueID)
	receipt.TeamKey = clipWebhookField(receipt.TeamKey)
	receipt.WorkspaceKey = clipWebhookField(receipt.WorkspaceKey)
	receipt.Outcome = clipWebhookField(receipt.Outcome)
	receipt.DeliveryKey = clipWebhookField(receipt.DeliveryKey)

	if err := tx.Create(&receipt).Error; err != nil {
		return uuid.Nil, err
	}
	return receipt.ID, nil
}

// LinearWebhookDeliveryAccepted reports whether this subscription already
// accepted the same Linear payload.
func LinearWebhookDeliveryAccepted(tx *gorm.DB, webhookID uuid.UUID, deliveryKey string) (bool, error) {
	deliveryKey = strings.TrimSpace(deliveryKey)
	if tx == nil || webhookID == uuid.Nil || deliveryKey == "" {
		return false, nil
	}

	var count int64
	err := tx.Model(&LinearWebhookReceipt{}).
		Where("webhook_id = ? AND delivery_key = ? AND outcome = ?", webhookID, deliveryKey, LinearWebhookOutcomeAccepted).
		Limit(1).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteExpiredLinearWebhookReceipts removes receipts older than olderThan.
// It deletes at most limit rows so one pass cannot scan the whole table.
func DeleteExpiredLinearWebhookReceipts(tx *gorm.DB, olderThan time.Time, limit int) (int64, error) {
	return deleteRowsLimited(tx, &LinearWebhookReceipt{}, limit, "received_at < ?", olderThan)
}

func UpdateLinearWebhookReceiptResult(tx *gorm.DB, id uuid.UUID, status int, outcome string, subscriptionCount int) error {
	if id == uuid.Nil {
		return nil
	}
	return tx.Model(&LinearWebhookReceipt{}).Where("id = ?", id).Updates(map[string]any{
		"http_status":        status,
		"outcome":            clipWebhookField(outcome),
		"subscription_count": subscriptionCount,
	}).Error
}

// AppendLinearWebhookTask records a SuperPlane task created from this webhook.
// A second append of the same task ID does not duplicate it.
func AppendLinearWebhookTask(tx *gorm.DB, receiptID, taskID uuid.UUID) error {
	if tx == nil || receiptID == uuid.Nil || taskID == uuid.Nil {
		return nil
	}

	var receipt LinearWebhookReceipt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", receiptID).First(&receipt).Error
	if err != nil {
		return err
	}

	ids := receipt.TaskIDList()
	if slices.Contains(ids, taskID.String()) {
		return nil
	}
	ids = append(ids, taskID.String())
	return tx.Model(&LinearWebhookReceipt{}).Where("id = ?", receiptID).Update("task_ids", strings.Join(ids, ",")).Error
}

func ListLinearWebhookReceipts(tx *gorm.DB, limit, offset int) ([]LinearWebhookReceipt, int64, error) {
	var total int64
	if err := tx.Model(&LinearWebhookReceipt{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var receipts []LinearWebhookReceipt
	err := tx.Order("received_at DESC").Limit(limit).Offset(offset).Find(&receipts).Error
	return receipts, total, err
}
