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

	spec := models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}

	retryStrategy := &models.RetryStrategy{
		Type: models.RetryStrategyTypeConstant,
		Constant: &models.ConstantRetryStrategy{
			MaxAttempts: 1,
			Delay:       interval,
		},
	}

	return c.execution.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, spec, retryStrategy)
}

func (c *ExecutionRequestContext) ScheduleActionWithRetry(actionName string, parameters map[string]any, interval time.Duration, maxAttempts int) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	retryStrategy := &models.RetryStrategy{
		Type: models.RetryStrategyTypeConstant,
		Constant: &models.ConstantRetryStrategy{
			MaxAttempts: maxAttempts,
			Delay:       interval,
		},
	}

	spec := models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}

	return c.execution.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, spec, retryStrategy)
}
