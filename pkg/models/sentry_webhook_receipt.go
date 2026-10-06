package models

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SentryWebhookOutcomeAccepted     = "accepted"
	SentryWebhookOutcomeNoConnection = "no_connection"
	SentryWebhookOutcomeRejected     = "rejected"
	SentryWebhookOutcomeFailed       = "failed"

	// SentryWebhookReceiptRetention is how long Installation Admin keeps a receipt.
	SentryWebhookReceiptRetention = 14 * 24 * time.Hour
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
	// TaskIDs lists SuperPlane tasks this webhook created. IDs are comma-separated.
	TaskIDs string
}

func (SentryWebhookReceipt) TableName() string {
	return "sentry_webhook_receipts"
}

func (r SentryWebhookReceipt) TaskIDList() []string {
	return splitCommaIDs(r.TaskIDs)
}

func CreateSentryWebhookReceipt(tx *gorm.DB, receipt SentryWebhookReceipt) (uuid.UUID, error) {
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
		return uuid.Nil, err
	}
	return receipt.ID, nil
}

// DeleteExpiredSentryWebhookReceipts removes receipts older than olderThan.
// It deletes at most limit rows so one pass cannot scan the whole table.
func DeleteExpiredSentryWebhookReceipts(tx *gorm.DB, olderThan time.Time, limit int) (int64, error) {
	return deleteRowsLimited(tx, &SentryWebhookReceipt{}, limit, "received_at < ?", olderThan)
}

func UpdateSentryWebhookReceiptResult(tx *gorm.DB, id uuid.UUID, status int, outcome string, integrationCount int) error {
	if id == uuid.Nil {
		return nil
	}
	return tx.Model(&SentryWebhookReceipt{}).Where("id = ?", id).Updates(map[string]any{
		"http_status":       status,
		"outcome":           clipWebhookField(outcome),
		"integration_count": integrationCount,
	}).Error
}

// AppendSentryWebhookTask records a SuperPlane task created from this webhook.
// A second append of the same task ID does not duplicate it.
func AppendSentryWebhookTask(tx *gorm.DB, receiptID, taskID uuid.UUID) error {
	if tx == nil || receiptID == uuid.Nil || taskID == uuid.Nil {
		return nil
	}

	var receipt SentryWebhookReceipt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", receiptID).First(&receipt).Error
	if err != nil {
		return err
	}

	ids := receipt.TaskIDList()
	if slices.Contains(ids, taskID.String()) {
		return nil
	}
	ids = append(ids, taskID.String())
	return tx.Model(&SentryWebhookReceipt{}).Where("id = ?", receiptID).Update("task_ids", strings.Join(ids, ",")).Error
}

// ListSentryWebhookReceipts returns one page of receipts, newest first.
// An empty project returns every receipt. A project value matches project_slug
// as a case-insensitive substring. Percent, underscore, and backslash stay literal.
func ListSentryWebhookReceipts(tx *gorm.DB, limit, offset int, project string) ([]SentryWebhookReceipt, int64, error) {
	var total int64
	if err := sentryWebhookReceiptQuery(tx, project).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var receipts []SentryWebhookReceipt
	err := sentryWebhookReceiptQuery(tx, project).
		Order("received_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&receipts).Error
	return receipts, total, err
}

func sentryWebhookReceiptQuery(tx *gorm.DB, project string) *gorm.DB {
	query := tx.Model(&SentryWebhookReceipt{})
	project = clipWebhookField(project)
	if project == "" {
		return query
	}
	return query.Where("project_slug ILIKE ? ESCAPE '\\'", containsLikePattern(project))
}

func clipWebhookField(value string) string {
	const maxBytes = 200
	value = strings.TrimSpace(value)
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}
