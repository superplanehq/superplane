package contexts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
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

	runAt := time.Now().Add(interval)
	request := models.CanvasNodeRequest{
		WorkflowID:  c.execution.WorkflowID,
		NodeID:      c.execution.NodeID,
		ExecutionID: &c.execution.ID,
		ID:          uuid.New(),
		State:       models.NodeExecutionRequestStatePending,
		Type:        models.NodeRequestTypeInvokeAction,
		Attempts:    0,
		RunAt:       runAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: actionName,
				Parameters: parameters,
			},
		}),
	}

	return c.tx.Create(&request).Error
}

func (c *ExecutionRequestContext) ScheduleActionWithRetry(actionName string, parameters map[string]any, interval time.Duration, maxAttempts int) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	runAt := time.Now().Add(interval)
	request := models.CanvasNodeRequest{
		WorkflowID:  c.execution.WorkflowID,
		NodeID:      c.execution.NodeID,
		ExecutionID: &c.execution.ID,
		ID:          uuid.New(),
		State:       models.NodeExecutionRequestStatePending,
		Type:        models.NodeRequestTypeInvokeAction,
		Attempts:    0,
		RunAt:       runAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: actionName,
				Parameters: parameters,
			},
		}),
	}

	if maxAttempts > 1 {
		request.RetryStrategy = datatypes.NewJSONType(models.RetryStrategy{
			Type: models.RetryStrategyTypeConstant,
			Constant: &models.ConstantRetryStrategy{
				MaxAttempts: maxAttempts,
				Delay:       int(interval / time.Second),
			},
		})
	}

	return c.tx.Create(&request).Error
}
