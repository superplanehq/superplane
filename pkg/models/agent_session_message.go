package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
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

// AgentSessionImage is a base64-encoded image attached to a user message.
type AgentSessionImage struct {
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AgentSessionMessage struct {
	ID              uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	SessionID       uuid.UUID
	ProviderEventID string
	Role            string
	Content         string
	ToolCallID      string
	ToolName        string
	ToolStatus      string
	Images          datatypes.JSONSlice[AgentSessionImage]
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
	if msg.Images == nil {
		msg.Images = datatypes.JSONSlice[AgentSessionImage]{}
	}

	if msg.ProviderEventID == "" {
		return tx.Create(msg).Error
	}

	// Preserve the existing content and tool name when the incoming event has
	// none. Anthropic tool_result events commonly omit both, and overwriting
	// with empty values hides the command/file that the user already saw on
	// tool_started.
	return tx.Exec(`
		INSERT INTO agent_session_messages
			(id, session_id, provider_event_id, role, content, tool_call_id, tool_name, tool_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (session_id, provider_event_id)
		WHERE provider_event_id <> ''
		DO UPDATE SET
			content = CASE
				WHEN agent_session_messages.tool_status IN ('finished', 'failed') AND EXCLUDED.tool_status = 'started'
				THEN agent_session_messages.content
				ELSE COALESCE(NULLIF(EXCLUDED.content, ''), agent_session_messages.content)
			END,
			tool_status = CASE
				WHEN agent_session_messages.tool_status IN ('finished', 'failed') AND EXCLUDED.tool_status = 'started'
				THEN agent_session_messages.tool_status
				ELSE EXCLUDED.tool_status
			END,
			tool_name = COALESCE(NULLIF(EXCLUDED.tool_name, ''), agent_session_messages.tool_name)
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

func FindAgentSessionMessage(tx *gorm.DB, id uuid.UUID) (*AgentSessionMessage, error) {
	var message AgentSessionMessage
	if err := tx.Where("id = ?", id).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

// ListAgentSessionMessagesPage returns up to `limit` messages strictly older
// than `before` (or the most recent `limit` when `before` is nil), in
// chronological order (oldest-first). Used for tail-paginated chat scroll.
func ListAgentSessionMessagesPage(sessionID uuid.UUID, before *AgentSessionMessage, limit int) ([]AgentSessionMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	query := database.Conn().
		Where("session_id = ?", sessionID).
		Order("created_at DESC, id DESC").
		Limit(limit)

	if before != nil && before.CreatedAt != nil {
		query = query.Where("(created_at, id) < (?, ?)", before.CreatedAt, before.ID)
	}

	var rows []AgentSessionMessage
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
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
