package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type RunStateContext struct {
	tx         *gorm.DB
	workflowID uuid.UUID
}

func NewRunStateContext(tx *gorm.DB, workflowID uuid.UUID) *RunStateContext {
	return &RunStateContext{tx: tx, workflowID: workflowID}
}

func (c *RunStateContext) HasActive() (bool, error) {
	return models.HasActiveCanvasRun(c.tx, c.workflowID)
}
