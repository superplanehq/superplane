package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	HeartbeatAt       *time.Time
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

func ListAgentSessionsForCanvasInTransaction(tx *gorm.DB, organizationID, canvasID uuid.UUID) ([]AgentSession, error) {
	var sessions []AgentSession
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("canvas_id = ?", canvasID).
		Find(&sessions).
		Error
	return sessions, err
}

func ListAgentSessionsForOrganizationInTransaction(tx *gorm.DB, organizationID uuid.UUID) ([]AgentSession, error) {
	var sessions []AgentSession
	err := tx.
		Where("organization_id = ?", organizationID).
		Find(&sessions).
		Error
	return sessions, err
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

func UpdateAgentSessionStatusIfUnchanged(sessionID uuid.UUID, status string, unchangedSince *time.Time) (bool, error) {
	return UpdateAgentSessionStatusIfUnchangedInTransaction(database.Conn(), sessionID, status, unchangedSince)
}

// TouchAgentSessionHeartbeat bumps heartbeat_at without touching status or
// updated_at, so the cleanup loop can distinguish "worker is still alive"
// from "row leaked". heartbeat_at is a dedicated column so cleanup can
// tell apart rows that have ever been heartbeated (tight cutoff) from
// rows owned by binaries that don't write heartbeats yet (loose cutoff)
// — critical during rolling deploys.
//
// Guarded on status='streaming' so a goroutine that survives past a
// reset (e.g. interrupt or another replica taking over) can't keep an
// already-idle row looking active. Uses UpdateColumn so GORM doesn't
// auto-bump updated_at — the worker's per-turn idle transition keys on
// updated_at as an optimistic concurrency check, and a heartbeat must
// not invalidate it.
func TouchAgentSessionHeartbeat(sessionID uuid.UUID) error {
	now := time.Now()
	return database.Conn().Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Where("status = ?", AgentSessionStatusStreaming).
		UpdateColumn("heartbeat_at", &now).
		Error
}

func UpdateAgentSessionProviderSessionInTransaction(tx *gorm.DB, sessionID uuid.UUID, providerSessionID, status string) error {
	now := time.Now()
	return tx.Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"provider_session_id": providerSessionID,
			"status":              status,
			"last_active_at":      &now,
			"updated_at":          &now,
		}).
		Error
}

func UpdateAgentSessionStatusIfUnchangedInTransaction(tx *gorm.DB, sessionID uuid.UUID, status string, unchangedSince *time.Time) (bool, error) {
	now := time.Now()
	query := tx.Model(&AgentSession{}).Where("id = ?", sessionID)
	if unchangedSince == nil {
		query = query.Where("updated_at IS NULL")
	} else {
		query = query.Where("updated_at = ?", *unchangedSince)
	}

	result := query.Updates(map[string]any{
		"status":         status,
		"last_active_at": &now,
		"updated_at":     &now,
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
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

func LockAgentSessionInTransaction(tx *gorm.DB, sessionID uuid.UUID) (*AgentSession, error) {
	var session AgentSession
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", sessionID).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FailStuckStreamingSessions marks any session in "streaming" state whose
// activity signal predates the corresponding cutoff as failed and returns
// the affected rows so the caller can fan out session_failed events.
//
// Two cutoffs because the activity signal is heterogeneous across binary
// versions: rows written by a heartbeat-aware worker carry a fresh
// heartbeat_at every tick (use heartbeatCutoff, tight); rows owned by an
// older binary leave heartbeat_at NULL, so cleanup falls back to
// updated_at with legacyCutoff (loose, sized above the max single turn).
// This keeps rolling deploys safe — upgraded pods running cleanup can't
// flag long-but-healthy turns held by not-yet-restarted pods.
func FailStuckStreamingSessions(heartbeatCutoff, legacyCutoff time.Time) ([]AgentSession, error) {
	var stuck []AgentSession
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("status = ?", AgentSessionStatusStreaming).
			Where("(heartbeat_at IS NOT NULL AND heartbeat_at < ?) OR (heartbeat_at IS NULL AND updated_at < ?)", heartbeatCutoff, legacyCutoff).
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
