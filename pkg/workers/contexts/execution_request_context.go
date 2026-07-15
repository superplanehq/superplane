package contexts

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionRequestContext struct {
	tx        *gorm.DB
	execution *models.CanvasNodeExecution
}

func NewExecutionRequestContext(tx *gorm.DB, execution *models.CanvasNodeExecution) *ExecutionRequestContext {
	return &ExecutionRequestContext{tx: tx, execution: execution}
}

func (c *ExecutionRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	finished, err := c.execution.IsFinished(c.tx)
	if err != nil {
		return err
	}

	if finished {
		return nil
	}

	runAt := time.Now().Add(interval)
	return c.execution.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}
