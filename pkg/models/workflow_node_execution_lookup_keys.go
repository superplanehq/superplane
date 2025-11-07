package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// WorkflowNodeExecutionLookupKey is a model that maps custom key-value pairs
// to specific workflow node executions, that enable fast lookups based on
// business logic identifiers.
//
// The guarantee is that the lookup is indexed and fast to query.
//

type WorkflowNodeExecutionLookupKey struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ExecutionID uuid.UUID `gorm:"type:uuid;not null"`
	Key         string    `gorm:"type:text;not null"`
	Value       string    `gorm:"type:text;not null"`
	CreatedAt   *time.Time
}

func CreateWorkflowNodeExecutionLookupKeyInTransaction(tx *gorm.DB, executionID uuid.UUID, key, value string) error {
	rec := WorkflowNodeExecutionLookupKey{
		ExecutionID: executionID,
		Key:         key,
		Value:       value,
	}

	return tx.Create(&rec).Error
}

func FindFirstWorkflowNodeExecutionLookupKeyInTransaction(tx *gorm.DB, key, value string) (*WorkflowNodeExecutionLookupKey, error) {
	var rec WorkflowNodeExecutionLookupKey

	err := tx.Where("key = ? AND value = ?", key, value).Order("created_at ASC").First(&rec).Error

	if err != nil {
		return nil, err
	}

	return &rec, nil
}
