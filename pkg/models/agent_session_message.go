package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	AgentMessageRoleUser      = "user"
	AgentMessageRoleAssistant = "assistant"
	AgentMessageRoleTool      = "tool"
	AgentMessageRoleSystem    = "system"

	AgentToolStatusStarted  = "started"
	AgentToolStatusFinished = "finished"
	AgentToolStatusFailed   = "failed"
)

type AgentSessionMessage struct {
	ID              uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	SessionID       uuid.UUID
	ProviderEventID string
	Role            string
	Content         string
	ToolCallID      string
	ToolName        string
	ToolStatus      string
	CreatedAt       *time.Time
}

func (AgentSessionMessage) TableName() string { return "agent_session_messages" }

// AppendAgentSessionMessageInTransaction upserts by (session_id,
// provider_event_id) when ProviderEventID is set, so a re-delivered provider
// event updates the existing row instead of duplicating it.
func AppendAgentSessionMessageInTransaction(tx *gorm.DB, msg *AgentSessionMessage) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.CreatedAt == nil {
		now := time.Now()
		msg.CreatedAt = &now
	}

	if msg.ProviderEventID == "" {
		return tx.Create(msg).Error
	}

	return tx.Exec(`
		INSERT INTO agent_session_messages
			(id, session_id, provider_event_id, role, content, tool_call_id, tool_name, tool_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (session_id, provider_event_id)
		WHERE provider_event_id <> ''
		DO UPDATE SET
			content = EXCLUDED.content,
			tool_status = EXCLUDED.tool_status,
			tool_name = EXCLUDED.tool_name
	`,
		msg.ID,
		msg.SessionID,
		msg.ProviderEventID,
		msg.Role,
		msg.Content,
		msg.ToolCallID,
		msg.ToolName,
		msg.ToolStatus,
		msg.CreatedAt,
	).Error
}

func AppendAgentSessionMessage(msg *AgentSessionMessage) error {
	return AppendAgentSessionMessageInTransaction(database.Conn(), msg)
}

func ListAgentSessionMessagesInTransaction(tx *gorm.DB, sessionID uuid.UUID) ([]AgentSessionMessage, error) {
	var messages []AgentSessionMessage
	err := tx.
		Where("session_id = ?", sessionID).
		Order("created_at ASC, id ASC").
		Find(&messages).
		Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func ListAgentSessionMessages(sessionID uuid.UUID) ([]AgentSessionMessage, error) {
	return ListAgentSessionMessagesInTransaction(database.Conn(), sessionID)
}

func CountAgentSessionMessagesInTransaction(tx *gorm.DB, sessionID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&AgentSessionMessage{}).Where("session_id = ?", sessionID).Count(&count).Error
	return count, err
}

// CloseOpenToolMessages flips lingering "started" tool rows to "finished"
// and returns them, so callers can broadcast the closure.
func CloseOpenToolMessages(sessionID uuid.UUID) ([]AgentSessionMessage, error) {
	var rows []AgentSessionMessage
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("session_id = ?", sessionID).
			Where("role = ?", AgentMessageRoleTool).
			Where("tool_status = ?", AgentToolStatusStarted).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return tx.Model(&AgentSessionMessage{}).
			Where("id IN ?", ids).
			Update("tool_status", AgentToolStatusFinished).Error
	})
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].ToolStatus = AgentToolStatusFinished
	}
	return rows, nil
}
