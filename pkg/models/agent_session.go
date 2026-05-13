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
	Title             string
	Status            string
	LastActiveAt      *time.Time
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	ArchivedAt        *time.Time
}

func (AgentSession) TableName() string { return "agent_sessions" }

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

func ListAgentSessionsForUser(organizationID, userID, canvasID uuid.UUID) ([]AgentSession, error) {
	var sessions []AgentSession
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
		Where("canvas_id = ?", canvasID).
		Where("archived_at IS NULL").
		Order("created_at DESC").
		Find(&sessions).
		Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
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

func UpdateAgentSessionTitleInTransaction(tx *gorm.DB, sessionID uuid.UUID, title string) error {
	now := time.Now()
	return tx.Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"title":      title,
			"updated_at": &now,
		}).
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

func ArchiveAgentSessionInTransaction(tx *gorm.DB, sessionID uuid.UUID) error {
	now := time.Now()
	return tx.Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Where("archived_at IS NULL").
		Updates(map[string]any{
			"archived_at": &now,
			"updated_at":  &now,
		}).
		Error
}

var ErrAgentSessionNotFound = errors.New("agent session not found")
