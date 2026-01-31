package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// CanvasNodeExecutionKV is a model that maps custom key-value pairs
// to specific canvas node executions, that enable fast lookups based on
// business logic identifiers.
//
// The guarantee is that the lookup is indexed and fast to query.
//
// DO NOT store any key/values here that are not strictly necessary for lookups,
// or fast retrieval of canvas node executions. Use the metadata field on
// CanvasNodeExecution for arbitrary data storage.
//

type CanvasNodeExecutionKV struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WorkflowID  uuid.UUID `gorm:"type:uuid;not null"`
	NodeID      string    `gorm:"type:varchar(128);not null"`
	ExecutionID uuid.UUID `gorm:"type:uuid;not null"`
	Key         string    `gorm:"type:text;not null"`
	Value       string    `gorm:"type:text;not null"`
	CreatedAt   *time.Time
}

func (c *CanvasNodeExecutionKV) TableName() string {
	return "workflow_node_execution_kvs"
}

func CreateNodeExecutionKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string, executionID uuid.UUID, key, value string) error {
	rec := CanvasNodeExecutionKV{
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		ExecutionID: executionID,
		Key:         key,
		Value:       value,
	}

	return tx.Create(&rec).Error
}

func FirstNodeExecutionByKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID, key, value string) (*CanvasNodeExecution, error) {
	var execution CanvasNodeExecution

	err := tx.
		Model(&CanvasNodeExecution{}).
		Where("id IN (?)", tx.
			Select("execution_id").
			Table("workflow_node_execution_kvs").
			Where("key = ? AND value = ?", key, value).
			Where("workflow_id = ?", workflowID).
			Where("node_id = ?", nodeID)).
		Order("created_at ASC").
		Limit(1).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}
