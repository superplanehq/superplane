package agents

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ChatSession represents a persisted agent chat session.
type ChatSession struct {
	ID                   string    `gorm:"column:id;primaryKey"`
	OrganizationID       string    `gorm:"column:organization_id"`
	UserID               string    `gorm:"column:user_id"`
	CanvasID             string    `gorm:"column:canvas_id"`
	AnthropicSessionID   string    `gorm:"column:anthropic_session_id"`
	CreatedAt            time.Time `gorm:"column:created_at"`
}

func (ChatSession) TableName() string { return "agent_sessions" }

// ChatMessage represents a stored message in a chat session.
type ChatMessage struct {
	ID         string    `gorm:"column:id;primaryKey"`
	SessionID  string    `gorm:"column:session_id"`
	Role       string    `gorm:"column:role"`
	Content    string    `gorm:"column:content"`
	ToolCallID string    `gorm:"column:tool_call_id"`
	ToolStatus string    `gorm:"column:tool_status"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (ChatMessage) TableName() string { return "agent_messages" }

// Store handles persistence for agent sessions and messages.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// FindSession finds an existing session for org/user/canvas.
func (s *Store) FindSession(orgID, userID, canvasID string) (*ChatSession, error) {
	var session ChatSession
	err := s.db.Where("organization_id = ? AND user_id = ? AND canvas_id = ?", orgID, userID, canvasID).First(&session).Error
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}
	return &session, nil
}

// CreateSession inserts a new session.
func (s *Store) CreateSession(orgID, userID, canvasID, anthropicSessionID string) (*ChatSession, error) {
	session := ChatSession{
		OrganizationID:     orgID,
		UserID:             userID,
		CanvasID:           canvasID,
		AnthropicSessionID: anthropicSessionID,
	}

	err := s.db.Raw(`
		INSERT INTO agent_sessions (organization_id, user_id, canvas_id, anthropic_session_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (organization_id, user_id, canvas_id) DO NOTHING
		RETURNING id, organization_id, user_id, canvas_id, anthropic_session_id, created_at
	`, orgID, userID, canvasID, anthropicSessionID).Scan(&session).Error
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// If ON CONFLICT hit, session.ID will be empty — fetch existing
	if session.ID == "" {
		return s.FindSession(orgID, userID, canvasID)
	}

	return &session, nil
}

// DeleteSession removes a session and its messages (CASCADE handles messages).
func (s *Store) DeleteSession(orgID, userID, canvasID string) error {
	return s.db.Where("organization_id = ? AND user_id = ? AND canvas_id = ?", orgID, userID, canvasID).Delete(&ChatSession{}).Error
}

// AppendMessage stores a message in the session.
func (s *Store) AppendMessage(sessionID, role, content, toolCallID, toolStatus string) error {
	return s.db.Exec(`
		INSERT INTO agent_messages (session_id, role, content, tool_call_id, tool_status)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
	`, sessionID, role, content, toolCallID, toolStatus).Error
}

// ListMessages returns all messages for a session, ordered by creation time.
func (s *Store) ListMessages(sessionID string) ([]ChatMessage, error) {
	var messages []ChatMessage
	err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return messages, nil
}
