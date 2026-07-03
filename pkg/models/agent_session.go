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
	ID                           uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID               uuid.UUID
	UserID                       uuid.UUID
	CanvasID                     uuid.UUID
	Provider                     string
	ProviderSessionID            string
	AgentToolSchemaRevision      string
	TrackedUsageInputTokens      int64
	TrackedUsageOutputTokens     int64
	TrackedUsageCacheReadTokens  int64
	TrackedUsageCacheWriteTokens int64
	TrackedUsageTotalTokens      int64
	TrackedUsageInitialized      bool `gorm:"default:true"`
	Status                       string
	LastActiveAt                 *time.Time
	HeartbeatAt                  *time.Time
	ContextReplayedAt            *time.Time
	CreatedAt                    *time.Time
	UpdatedAt                    *time.Time
}

func (AgentSession) TableName() string { return "agent_sessions" }

var ErrAgentSessionNotFound = errors.New("agent session not found")

type AgentSessionTokenUsage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalTokens      int64
}

func (u AgentSessionTokenUsage) HasUsage() bool {
	return u.TotalTokens > 0
}

func CreateAgentSessionInTransaction(tx *gorm.DB, session *AgentSession) error {
	session.TrackedUsageInitialized = true
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
			// Clear so a new streaming turn starts in the legacy-cutoff
			// branch until the worker writes its first heartbeat.
			"heartbeat_at": gorm.Expr("NULL"),
		}).
		Error
}

func UpdateAgentSessionStatus(sessionID uuid.UUID, status string) error {
	return UpdateAgentSessionStatusInTransaction(database.Conn(), sessionID, status)
}

func UpdateAgentSessionStatusIfUnchanged(sessionID uuid.UUID, status string, unchangedSince *time.Time) (bool, error) {
	return UpdateAgentSessionStatusIfUnchangedInTransaction(database.Conn(), sessionID, status, unchangedSince)
}

func IsAgentSessionStreaming(sessionID uuid.UUID) (bool, error) {
	var count int64
	if err := database.Conn().Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Where("status = ?", AgentSessionStatusStreaming).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// TouchAgentSessionHeartbeat uses UpdateColumn so updated_at stays put —
// it's the optimistic-concurrency key for the per-turn idle transition.
func TouchAgentSessionHeartbeat(sessionID uuid.UUID) error {
	now := time.Now()
	return database.Conn().Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Where("status = ?", AgentSessionStatusStreaming).
		UpdateColumn("heartbeat_at", &now).
		Error
}

func MarkAgentSessionContextReplayed(sessionID uuid.UUID) error {
	now := time.Now()
	return database.Conn().Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"context_replayed_at": &now,
			"updated_at":          &now,
		}).
		Error
}

func UpdateAgentSessionProviderSessionInTransaction(tx *gorm.DB, sessionID uuid.UUID, providerSessionID, toolSchemaRevision, status string) error {
	now := time.Now()
	return tx.Model(&AgentSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"provider_session_id":              providerSessionID,
			"agent_tool_schema_revision":       toolSchemaRevision,
			"tracked_usage_input_tokens":       0,
			"tracked_usage_output_tokens":      0,
			"tracked_usage_cache_read_tokens":  0,
			"tracked_usage_cache_write_tokens": 0,
			"tracked_usage_total_tokens":       0,
			"tracked_usage_initialized":        true,
			"status":                           status,
			"last_active_at":                   &now,
			"updated_at":                       &now,
			"heartbeat_at":                     gorm.Expr("NULL"),
			"context_replayed_at":              gorm.Expr("NULL"),
		}).Error
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
		"heartbeat_at":   gorm.Expr("NULL"),
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

// DeleteAgentSessionForUserCanvas takes an explicit *gorm.DB per the transaction
// guidelines (no InTransaction suffix / conn wrapper); pass a tx or database.DB(ctx).
func DeleteAgentSessionForUserCanvas(tx *gorm.DB, organizationID, userID, canvasID uuid.UUID) error {
	return tx.
		Where("organization_id = ?", organizationID).
		Where("user_id = ?", userID).
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

func CalculateAgentSessionTokenUsageDelta(sessionID uuid.UUID, usage AgentSessionTokenUsage) (AgentSessionTokenUsage, bool, error) {
	session, err := FindAgentSession(sessionID)
	if err != nil {
		return AgentSessionTokenUsage{}, false, err
	}
	if !session.TrackedUsageInitialized {
		return AgentSessionTokenUsage{}, false, nil
	}

	return agentSessionTokenUsageDelta(session, usage), true, nil
}

func MarkAgentSessionTokenUsageTracked(sessionID uuid.UUID, usage AgentSessionTokenUsage) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		session, err := LockAgentSessionInTransaction(tx, sessionID)
		if err != nil {
			return err
		}

		return tx.Model(&AgentSession{}).
			Where("id = ?", sessionID).
			Updates(map[string]any{
				"tracked_usage_input_tokens":       maxInt64(session.TrackedUsageInputTokens, usage.InputTokens),
				"tracked_usage_output_tokens":      maxInt64(session.TrackedUsageOutputTokens, usage.OutputTokens),
				"tracked_usage_cache_read_tokens":  maxInt64(session.TrackedUsageCacheReadTokens, usage.CacheReadTokens),
				"tracked_usage_cache_write_tokens": maxInt64(session.TrackedUsageCacheWriteTokens, usage.CacheWriteTokens),
				"tracked_usage_total_tokens":       maxInt64(session.TrackedUsageTotalTokens, usage.TotalTokens),
				"tracked_usage_initialized":        true,
			}).Error
	})
}

// FailStuckStreamingSessions flags leaked streaming rows. Heartbeated rows
// use heartbeatCutoff (tight); rows with no heartbeat yet — pre-heartbeat
// binaries, or new turns before the worker's first tick — fall back to
// updated_at with legacyCutoff (loose, sized above agentStreamTimeout).
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
				"status":       AgentSessionStatusFailed,
				"updated_at":   &now,
				"heartbeat_at": gorm.Expr("NULL"),
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return stuck, nil
}

func agentSessionTokenUsageDelta(session *AgentSession, usage AgentSessionTokenUsage) AgentSessionTokenUsage {
	delta := AgentSessionTokenUsage{
		InputTokens:      nonNegativeDelta(usage.InputTokens, session.TrackedUsageInputTokens),
		OutputTokens:     nonNegativeDelta(usage.OutputTokens, session.TrackedUsageOutputTokens),
		CacheReadTokens:  nonNegativeDelta(usage.CacheReadTokens, session.TrackedUsageCacheReadTokens),
		CacheWriteTokens: nonNegativeDelta(usage.CacheWriteTokens, session.TrackedUsageCacheWriteTokens),
		TotalTokens:      nonNegativeDelta(usage.TotalTokens, session.TrackedUsageTotalTokens),
	}
	if delta.TotalTokens == 0 {
		delta.TotalTokens = delta.InputTokens + delta.OutputTokens + delta.CacheReadTokens + delta.CacheWriteTokens
	}

	return delta
}

func nonNegativeDelta(current, tracked int64) int64 {
	if current <= tracked {
		return 0
	}

	return current - tracked
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}
