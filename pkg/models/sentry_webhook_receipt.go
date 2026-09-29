package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SentryWebhookOutcomeAccepted     = "accepted"
	SentryWebhookOutcomeNoConnection = "no_connection"
	SentryWebhookOutcomeRejected     = "rejected"
	SentryWebhookOutcomeFailed       = "failed"

	sentryWebhookReceiptRetention = 14 * 24 * time.Hour
)

// SentryWebhookReceipt is one incoming Sentry webhook. It stores the event
// type and the result. It does not store the payload, signature, or tokens.
type SentryWebhookReceipt struct {
	ID               uuid.UUID `gorm:"primaryKey"`
	ReceivedAt       time.Time
	HookResource     string
	Action           string
	InstallationUUID string
	OrganizationSlug string
	ProjectSlug      string
	IssueID          string
	IssueShortID     string
	HTTPStatus       int
	Outcome          string
	IntegrationCount int
}

func (SentryWebhookReceipt) TableName() string {
	return "sentry_webhook_receipts"
}

func CreateSentryWebhookReceipt(tx *gorm.DB, receipt SentryWebhookReceipt) error {
	if receipt.ID == uuid.Nil {
		receipt.ID = uuid.New()
	}
	if receipt.ReceivedAt.IsZero() {
		receipt.ReceivedAt = time.Now().UTC()
	}
	receipt.HookResource = clipWebhookField(receipt.HookResource)
	receipt.Action = clipWebhookField(receipt.Action)
	receipt.InstallationUUID = clipWebhookField(receipt.InstallationUUID)
	receipt.OrganizationSlug = clipWebhookField(receipt.OrganizationSlug)
	receipt.ProjectSlug = clipWebhookField(receipt.ProjectSlug)
	receipt.IssueID = clipWebhookField(receipt.IssueID)
	receipt.IssueShortID = clipWebhookField(receipt.IssueShortID)
	receipt.Outcome = clipWebhookField(receipt.Outcome)

	if err := tx.Create(&receipt).Error; err != nil {
		return err
	}

	cutoff := time.Now().UTC().Add(-sentryWebhookReceiptRetention)
	return tx.Where("received_at < ?", cutoff).Delete(&SentryWebhookReceipt{}).Error
}

func ListSentryWebhookReceipts(tx *gorm.DB, limit, offset int) ([]SentryWebhookReceipt, int64, error) {
	var total int64
	if err := tx.Model(&SentryWebhookReceipt{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var receipts []SentryWebhookReceipt
	err := tx.Order("received_at DESC").Limit(limit).Offset(offset).Find(&receipts).Error
	return receipts, total, err
}

func clipWebhookField(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 200 {
		return value
	}
	return value[:200]
}
