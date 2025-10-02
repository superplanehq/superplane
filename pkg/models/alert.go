package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	AlertTypeError   = "error"
	AlertTypeWarning = "warning"
	AlertTypeInfo    = "info"
)

type Alert struct {
	ID             uuid.UUID  `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID       uuid.UUID  `gorm:"column:canvas_id"`
	SourceID       uuid.UUID  `gorm:"column:source_id"`
	SourceType     string     `gorm:"column:source_type"`
	Message        string     `gorm:"column:message"`
	Acknowledged   bool       `gorm:"column:acknowledged"`
	AcknowledgedAt *time.Time `gorm:"column:acknowledged_at"`
	Type           string     `gorm:"column:type"`
	CreatedAt      *time.Time `gorm:"column:created_at"`
}

func NewAlert(canvasID uuid.UUID, sourceID uuid.UUID, sourceType string, message string, alertType string) (*Alert, error) {
	now := time.Now()
	return &Alert{
		CanvasID:   canvasID,
		SourceID:   sourceID,
		SourceType: sourceType,
		Message:    message,
		Type:       alertType,
		CreatedAt:  &now,
	}, nil
}

func (a *Alert) Acknowledge() {
	now := time.Now()
	a.Acknowledged = true
	a.AcknowledgedAt = &now
}

func (a *Alert) Create() error {
	return a.CreateInTransaction(database.Conn())
}

func (a *Alert) CreateInTransaction(tx *gorm.DB) error {
	return tx.Create(a).Error
}

func (a *Alert) Update() error {
	return a.UpdateInTransaction(database.Conn())
}

func (a *Alert) UpdateInTransaction(tx *gorm.DB) error {
	return tx.Save(a).Error
}

func ListAlerts(canvasID uuid.UUID, includeAcknowledged bool, before *time.Time) ([]Alert, error) {
	var alerts []Alert

	query := database.Conn().
		Where("canvas_id = ?", canvasID).
		Order("created_at DESC")

	if !includeAcknowledged {
		query = query.Where("acknowledged = false")
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	err := query.Find(&alerts).Error

	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func FindAlertByID(alertID uuid.UUID, canvasID uuid.UUID) (*Alert, error) {
	var alert Alert
	err := database.Conn().
		Where("id = ? AND canvas_id = ?", alertID, canvasID).
		First(&alert).Error

	if err != nil {
		return nil, err
	}

	return &alert, nil
}
