package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PlanningSessionActivityStatusRunning     = "running"
	PlanningSessionActivityStatusPassed      = "passed"
	PlanningSessionActivityStatusFailed      = "failed"
	PlanningSessionActivityStatusCancelled   = "cancelled"
	PlanningSessionActivityStatusTimedOut    = "timed_out"
	PlanningSessionActivityStatusInterrupted = "interrupted"
)

type PlanningSessionActivityOutput struct {
	Stream string `json:"stream,omitempty"`
	Text   string `json:"text,omitempty"`
}

type PlanningSessionActivityItem struct {
	Type          string                          `json:"type"`
	ID            string                          `json:"id"`
	Kind          string                          `json:"kind,omitempty"`
	Text          string                          `json:"text,omitempty"`
	Name          string                          `json:"name,omitempty"`
	Input         string                          `json:"input,omitempty"`
	Output        string                          `json:"output,omitempty"`
	OutputStreams []PlanningSessionActivityOutput `json:"output_streams,omitempty"`
	Status        string                          `json:"status,omitempty"`
	Code          string                          `json:"code,omitempty"`
	StartedAt     int64                           `json:"started_at,omitempty"`
	DurationMs    int64                           `json:"duration_ms,omitempty"`
	ExitCode      *int                            `json:"exit_code,omitempty"`
	Signal        string                          `json:"signal,omitempty"`
	Truncated     bool                            `json:"truncated,omitempty"`
}

type PlanningSessionActivitySnapshot struct {
	SchemaVersion int                           `json:"schema_version"`
	ActivityID    string                        `json:"activity_id"`
	Provider      string                        `json:"provider"`
	Turn          int                           `json:"turn"`
	Sequence      int64                         `json:"sequence"`
	Status        string                        `json:"status"`
	StartedAt     int64                         `json:"started_at"`
	CompletedAt   *int64                        `json:"completed_at,omitempty"`
	Items         []PlanningSessionActivityItem `json:"items"`
	Truncated     bool                          `json:"truncated,omitempty"`
}

type PlanningSessionActivity struct {
	ID            uuid.UUID
	SessionID     uuid.UUID
	SchemaVersion int
	Provider      string
	Status        string
	LastSequence  int64
	Snapshot      datatypes.JSONType[PlanningSessionActivitySnapshot]
	StartedAt     time.Time
	CompletedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (PlanningSessionActivity) TableName() string {
	return "factory_planning_session_activities"
}

func (s *FactoryPlanningSession) UpsertActivity(tx *gorm.DB, activity PlanningSessionActivity) error {
	if activity.ID == uuid.Nil || activity.SessionID != s.ID || activity.SchemaVersion < 1 || activity.LastSequence < 0 {
		return fmt.Errorf("%w: invalid activity", ErrFactoryPlanningSessionInvalid)
	}

	var existing PlanningSessionActivity
	err := tx.Select("id", "session_id").Where("id = ?", activity.ID).First(&existing).Error
	if err == nil && existing.SessionID != s.ID {
		return fmt.Errorf("%w: activity belongs to a different session", ErrFactoryPlanningSessionInvalid)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"schema_version", "provider", "status", "last_sequence", "snapshot", "started_at", "completed_at", "updated_at",
		}),
		Where: clause.Where{Exprs: []clause.Expression{clause.Expr{
			SQL: "factory_planning_session_activities.last_sequence < EXCLUDED.last_sequence",
		}}},
	}).Create(&activity).Error
}

func ListPlanningSessionActivities(tx *gorm.DB, sessionID uuid.UUID) ([]PlanningSessionActivity, error) {
	var activities []PlanningSessionActivity
	err := tx.Where("session_id = ?", sessionID).Order("started_at ASC, id ASC").Find(&activities).Error
	return activities, err
}

func FindPlanningSessionActivity(tx *gorm.DB, sessionID, activityID uuid.UUID) (*PlanningSessionActivity, error) {
	var activity PlanningSessionActivity
	err := tx.Where("session_id = ? AND id = ?", sessionID, activityID).First(&activity).Error
	return &activity, err
}
