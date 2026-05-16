package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	AgentSessionStatusIdle       = "idle"
	AgentSessionStatusStreaming  = "streaming"
	AgentSessionStatusFailed     = "failed"
	AgentSessionStatusTerminated = "terminated"
)

type AgentSession struct {
	ID                uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID    uuid.UUID
	UserID            uuid.UUID
	CanvasID          uuid.UUID
	Provider          string
	ProviderSessionID string
	Status            string
	LastActiveAt      *time.Time
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

func (AgentSession) TableName() string { return "agent_sessions" }

var ErrAgentSessionNotFound = errors.New("agent session not found")

func CreateAgentSessionInTransaction(tx *gorm.DB, session *AgentSession) error {
	return tx.Create(session).Error
}

func FindAgentSessionInTransaction(tx *gorm.DB, sessionID uuid.UUID) (*AgentSession, error) {
	var session AgentSession
	err := tx.Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func FindAgentSession(sessionID uuid.UUID) (*AgentSession, error) {
	return FindAgentSessionInTransaction(database.Conn(), sessionID)
}

// FindAgentSessionForUserInTransaction enforces ownership: sessions are
// private per user, so a session is invisible to anyone but its creator.
func FindAgentSessionForUserInTransaction(tx *gorm.DB, organizationID, userID, sessionID uuid.UUID) (*AgentSession, error) {
	var session AgentSession
	err := tx.
		Where("id = ?", sessionID).
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func FindAgentSessionForUser(organizationID, userID, sessionID uuid.UUID) (*AgentSession, error) {
	return FindAgentSessionForUserInTransaction(database.Conn(), organizationID, userID, sessionID)
}

func FindAgentSessionByCanvasInTransaction(tx *gorm.DB, organizationID, userID, canvasID uuid.UUID) (*AgentSession, error) {
	var session AgentSession
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id = ?", canvasID).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func UpdateAgentSessionStatusInTransaction(tx *gorm.DB, sessionID uuid.UUID, status string) error {
	now := time.Now()
	return tx.Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"status":         status,
			"last_active_at": &now,
			"updated_at":     &now,
		}).
		Error
}

func UpdateAgentSessionStatus(sessionID uuid.UUID, status string) error {
	return UpdateAgentSessionStatusInTransaction(database.Conn(), sessionID, status)
}

func DeleteAgentSessionsForCanvasInTransaction(tx *gorm.DB, organizationID, canvasID uuid.UUID) error {
	return tx.
		Where("organization_id = ?", organizationID).
		Where("canvas_id = ?", canvasID).
		Delete(&AgentSession{}).
		Error
}

func DeleteAgentSessionsForOrganizationInTransaction(tx *gorm.DB, organizationID uuid.UUID) error {
	return tx.
		Where("organization_id = ?", organizationID).
		Delete(&AgentSession{}).
		Error
}

// FailStuckStreamingSessions marks any session in "streaming" state whose
// last update predates cutoff as failed and returns the affected rows so
// the caller can fan out session_failed events.
func FailStuckStreamingSessions(cutoff time.Time) ([]AgentSession, error) {
	var stuck []AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("status = ?", AgentSessionStatusStreaming).
			Where("updated_at < ?", cutoff).
			Find(&stuck).Error; err != nil {
			return err
		}
		if len(stuck) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(stuck))
		for _, s := range stuck {
			ids = append(ids, s.ID)
		}
		now := time.Now()
		return tx.Model(&AgentSession{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":     AgentSessionStatusFailed,
				"updated_at": &now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return stuck, nil
}
