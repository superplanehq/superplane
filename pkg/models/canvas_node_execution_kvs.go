package models

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

// FindLatestNodeExecutionKVValueInTransaction returns the value from the newest
// row for this execution and key (by created_at, then id).
func FindLatestNodeExecutionKVValueInTransaction(tx *gorm.DB, executionID uuid.UUID, key string) (string, error) {
	var rec CanvasNodeExecutionKV
	err := tx.
		Where("execution_id = ? AND key = ?", executionID, key).
		Order("created_at DESC, id DESC").
		Limit(1).
		First(&rec).Error
	if err != nil {
		return "", err
	}
	return rec.Value, nil
}

// FirstNodeExecutionByKVInTransaction resolves the execution a key/value
// pair points at, e.g. an external correlation ID stored by a component.
// Values are expected to identify exactly one execution within a node.
// When more than one execution matches, the oldest wins — preserved
// behavior — and an error is logged, because a collision means a
// component stored a value that is not unique per execution and webhooks
// or integration messages may be routed to the wrong execution.
func FirstNodeExecutionByKVInTransaction(tx *gorm.DB, workflowID uuid.UUID, nodeID, key, value string) (*CanvasNodeExecution, error) {
	var executionIDs []uuid.UUID
	err := tx.
		Table("workflow_node_execution_kvs").
		Distinct("execution_id").
		Where("key = ? AND value = ?", key, value).
		Where("workflow_id = ?", workflowID).
		Where("node_id = ?", nodeID).
		Pluck("execution_id", &executionIDs).
		Error
	if err != nil {
		return nil, err
	}

	if len(executionIDs) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	if len(executionIDs) > 1 {
		log.Errorf(
			"KV %s=%s on node %s in workflow %s matches %d executions — values must be unique per execution; using the oldest",
			key, value, nodeID, workflowID, len(executionIDs),
		)
	}

	var execution CanvasNodeExecution
	err = tx.
		Where("workflow_id = ?", workflowID).
		Where("id IN ?", executionIDs).
		Order("created_at ASC").
		First(&execution).
		Error
	if err != nil {
		return nil, err
	}

	return &execution, nil
}
