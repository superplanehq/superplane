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
	WorkflowID  uuid.UUID `gorm:"type:uuid;not null"`
	NodeID      string    `gorm:"type:varchar(128);not null"`
	ExecutionID uuid.UUID `gorm:"type:uuid;not null"`
	Key         string    `gorm:"type:text;not null"`
	Value       string    `gorm:"type:text;not null"`
	CreatedAt   *time.Time
}

func CreateWorkflowNodeExecutionKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID string, executionID uuid.UUID, key, value string) error {
	rec := WorkflowNodeExecutionKV{
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		ExecutionID: executionID,
		Key:         key,
		Value:       value,
	}

	return tx.Create(&rec).Error
}

func FindNodeExecutionByKVInTransaction(tx *gorm.DB, workflowID *uuid.UUID, nodeID, key, value string) (*WorkflowNodeExecution, error) {
	var execution WorkflowNodeExecution

	err := tx.
		Joins("JOIN workflow_node_execution_kvs AS kvs ON workflow_node_executions.id = workflow_node_execution_kvs.execution_id").
		Where("kvs.key = ? AND kvs.value = ?", key, value).
		Where("kvs.workflow_id = ?", *workflowID).
		Where("kvs.node_id = ?", nodeID).
		Order("kvs.created_at ASC").
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}
