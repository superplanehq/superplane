package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AppMessage struct {
	ID        uuid.UUID
	CanvasID  uuid.UUID
	NodeID    string
	Payload   datatypes.JSON
	CreatedAt time.Time
}

func (m *AppMessage) TableName() string {
	return "app_messages"
}

func CreateAppMessage(tx *gorm.DB, canvasID uuid.UUID, nodeID string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	message := &AppMessage{
		ID:        uuid.New(),
		CanvasID:  canvasID,
		NodeID:    nodeID,
		Payload:   datatypes.JSON(payloadBytes),
		CreatedAt: time.Now(),
	}

	return tx.Create(message).Error
}

func ListAppMessages() ([]AppMessage, error) {
	var messages []AppMessage

	query := database.Conn().
		Table("app_messages").
		Select("app_messages.*")

	err := withActiveCanvas(query, "app_messages.canvas_id").
		Order("app_messages.created_at ASC").
		Find(&messages).
		Error
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func LockAppMessage(tx *gorm.DB, id uuid.UUID) (*AppMessage, error) {
	var message AppMessage

	query := tx.
		Table("app_messages").
		Select("app_messages.*").
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Table:    clause.Table{Name: "app_messages"},
			Options:  "SKIP LOCKED",
		}).
		Where("app_messages.id = ?", id)

	err := withActiveCanvas(query, "app_messages.canvas_id").
		First(&message).
		Error
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (m *AppMessage) Delete(tx *gorm.DB) error {
	return tx.Delete(m).Error
}
