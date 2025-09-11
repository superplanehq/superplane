package models

import (
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	EventRejectionReasonFiltered = "filtered"
	EventRejectionReasonError    = "error"
)

type EventRejection struct {
	ID            uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	EventID       uuid.UUID
	ComponentType string
	ComponentID   uuid.UUID
	Reason        string
	Message       string
	RejectedAt    *time.Time
}

func RejectEventInTransaction(tx *gorm.DB, eventID, componentID uuid.UUID, componentType, reason, message string) error {
	now := time.Now()
	rejection := &EventRejection{
		EventID:       eventID,
		ComponentType: componentType,
		ComponentID:   componentID,
		Reason:        reason,
		Message:       message,
		RejectedAt:    &now,
	}

	return tx.Create(rejection).Error
}

func ListEventRejections(componentType string, componentID uuid.UUID) ([]EventRejection, error) {
	query := database.Conn().
		Where("component_type = ?", componentType).
		Where("component_id = ?", componentID)

	var rejections []EventRejection
	err := query.Find(&rejections).Error
	if err != nil {
		return nil, err
	}

	return rejections, nil
}
