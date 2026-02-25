package contexts

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type CanvasDataContext struct {
	tx         *gorm.DB
	workflowID uuid.UUID
}

func NewCanvasDataContext(tx *gorm.DB, workflowID uuid.UUID) *CanvasDataContext {
	return &CanvasDataContext{tx: tx, workflowID: workflowID}
}

func (c *CanvasDataContext) Set(key string, value any) error {
	return models.UpsertCanvasDataKVInTransaction(c.tx, c.workflowID, key, value)
}

func (c *CanvasDataContext) Get(key string) (any, bool, error) {
	record, err := models.FindCanvasDataKVInTransaction(c.tx, c.workflowID, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return record.Value.Data(), true, nil
}

func (c *CanvasDataContext) List() (map[string]any, error) {
	records, err := models.ListCanvasDataKVsInTransaction(c.tx, c.workflowID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(records))
	for _, record := range records {
		result[record.Key] = record.Value.Data()
	}

	return result, nil
}
