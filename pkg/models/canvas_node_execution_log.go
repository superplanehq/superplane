package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CanvasNodeExecutionLogTypeLine     = "line"
	CanvasNodeExecutionLogTypeError    = "error"
	CanvasNodeExecutionLogTypeCmdStart = "cmd_start"
	CanvasNodeExecutionLogTypeCmdEnd   = "cmd_end"
)

type CanvasNodeExecutionLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	WorkflowID   uuid.UUID
	RunID        uuid.UUID
	NodeID       string
	ExecutionID  uuid.UUID
	Sequence     int64
	Type         string
	Text         *string
	Message      *string
	CommandIndex *int
	Status       *string
	DurationMs   *int64
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
}

func (l *CanvasNodeExecutionLog) TableName() string {
	return "workflow_node_execution_logs"
}

func CreateNodeExecutionLogs(logs []CanvasNodeExecutionLog) error {
	return CreateNodeExecutionLogsInTransaction(database.Conn(), logs)
}

func ListNodeExecutionLogs(workflowID, executionID uuid.UUID, limit int, afterSequence *int64) ([]CanvasNodeExecutionLog, error) {
	return ListNodeExecutionLogsInTransaction(database.Conn(), workflowID, executionID, limit, afterSequence)
}

func CreateNodeExecutionLogsInTransaction(tx *gorm.DB, logs []CanvasNodeExecutionLog) error {
	if len(logs) == 0 {
		return nil
	}

	return tx.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "execution_id"}, {Name: "sequence"}},
			DoNothing: true,
		}).
		Create(&logs).
		Error
}

func ListNodeExecutionLogsInTransaction(tx *gorm.DB, workflowID, executionID uuid.UUID, limit int, afterSequence *int64) ([]CanvasNodeExecutionLog, error) {
	var logs []CanvasNodeExecutionLog
	query := tx.
		Where("workflow_id = ?", workflowID).
		Where("execution_id = ?", executionID).
		Order("sequence ASC").
		Limit(limit)

	if afterSequence != nil {
		query = query.Where("sequence > ?", *afterSequence)
	}

	err := query.Find(&logs).Error
	if err != nil {
		return nil, err
	}

	return logs, nil
}
