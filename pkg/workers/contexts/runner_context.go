package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type RunnerContext struct {
	tx             *gorm.DB
	organizationID uuid.UUID
	execution      *models.CanvasNodeExecution
}

func NewRunnerContext(tx *gorm.DB, organizationID uuid.UUID, execution *models.CanvasNodeExecution) *RunnerContext {
	return &RunnerContext{
		tx:             tx,
		organizationID: organizationID,
		execution:      execution,
	}
}

func (c *RunnerContext) ExecuteCode(code string, timeout int) error {
	_, err := models.CreateExecuteCodeJob(c.tx, c.organizationID, c.execution.ID, &models.RunnerJobSpec{
		ExecuteCode: &models.ExecuteCodeJobSpec{
			Code:    code,
			Timeout: timeout,
		},
	})
	return err
}
