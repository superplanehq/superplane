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
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	EventID    uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Reason     string
	Message    string
	RejectedAt *time.Time

	Event *Event `gorm:"foreignKey:EventID;references:ID"`
}

func RejectEvent(eventID, targetID uuid.UUID, targetType, reason, message string) (*EventRejection, error) {
	return RejectEventInTransaction(database.Conn(), eventID, targetID, targetType, reason, message)
}

func RejectEventInTransaction(tx *gorm.DB, eventID, targetID uuid.UUID, targetType, reason, message string) (*EventRejection, error) {
	now := time.Now()
	rejection := &EventRejection{
		EventID:    eventID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		Message:    message,
		RejectedAt: &now,
	}

	err := tx.Create(rejection).Error
	if err != nil {
		return nil, err
	}

	return rejection, nil
}

func FindEventRejectionByID(id uuid.UUID) (*EventRejection, error) {
	var rejection EventRejection
	err := database.Conn().
		Preload("Event").
		Where("id = ?", id).
		First(&rejection).Error
	if err != nil {
		return nil, err
	}

	return &rejection, nil
}

func ListEventRejections(targetType string, targetID uuid.UUID) ([]EventRejection, error) {
	query := database.Conn().
		Preload("Event").
		Where("target_type = ?", targetType).
		Where("target_id = ?", targetID)

	var rejections []EventRejection
	err := query.Find(&rejections).Error
	if err != nil {
		return nil, err
	}

	return rejections, nil
}

func FilterEventRejections(targetType string, targetID uuid.UUID, limit int, before *time.Time) ([]EventRejection, error) {
	var rejections []EventRejection
	query := database.Conn().
		Preload("Event").
		Where("target_type = ?", targetType).
		Where("target_id = ?", targetID)

	if before != nil {
		query = query.Where("rejected_at < ?", before)
	}

	query = query.Order("rejected_at DESC").Limit(limit)
	err := query.Find(&rejections).Error
	if err != nil {
		return nil, err
	}

	return rejections, nil
}

func CountEventRejections(targetType string, targetID uuid.UUID) (int64, error) {
	query := database.Conn().
		Model(&EventRejection{}).
		Where("target_type = ?", targetType).
		Where("target_id = ?", targetID)

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
