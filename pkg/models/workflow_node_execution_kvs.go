package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// WorkflowNodeExecutionKV is a model that maps custom key-value pairs
// to specific workflow node executions, that enable fast lookups based on
// business logic identifiers.
//
// The guarantee is that the lookup is indexed and fast to query.
//
// DO NOT store any key/values here that are not strictly necessary for lookups,
// or fast retrieval of workflow node executions. Use the metadata field on
// WorkflowNodeExecution for arbitrary data storage.
//

type WorkflowNodeExecutionKV struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExecutionID uuid.UUID `gorm:"type:uuid;not null"`
	Key         string    `gorm:"type:text;not null"`
	Value       string    `gorm:"type:text;not null"`
	CreatedAt   *time.Time
}

func (WorkflowNodeExecutionKV) TableName() string { return "workflow_node_execution_kvs" }

func CreateWorkflowNodeExecutionKVInTransaction(tx *gorm.DB, executionID uuid.UUID, key, value string) error {
	rec := WorkflowNodeExecutionKV{
		ExecutionID: executionID,
		Key:         key,
		Value:       value,
	}

	return tx.Create(&rec).Error
}

func FindFirstWorkflowNodeExecutionKVInTransaction(tx *gorm.DB, key, value string) (*WorkflowNodeExecutionKV, error) {
	var rec WorkflowNodeExecutionKV

	err := tx.Where("key = ? AND value = ?", key, value).Order("created_at ASC").First(&rec).Error

	if err != nil {
		return nil, err
	}

	return &rec, nil
}
