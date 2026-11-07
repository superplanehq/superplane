package contexts

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ExecutionRequestContext struct {
	tx             *gorm.DB
	execution      *models.CanvasNodeExecution
	currentRequest *models.CanvasNodeRequest
}

func NewExecutionRequestContext(tx *gorm.DB, execution *models.CanvasNodeExecution) *ExecutionRequestContext {
	return &ExecutionRequestContext{tx: tx, execution: execution}
}

func (c *ExecutionRequestContext) WithCurrentRequest(request *models.CanvasNodeRequest) *ExecutionRequestContext {
	c.currentRequest = request
	return c
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

	if parameters == nil {
		parameters = map[string]any{}
	}

	runAt := time.Now().Add(interval)
	createdBefore := runAt
	if c.currentRequest != nil {
		// A newer webhook must retain its earlier deadline when an in-flight hook schedules its next poll.
		createdBefore = c.currentRequest.CreatedAt
	}
	runAt, err = models.CompletePendingExecutionActionRequests(c.tx, c.execution.ID, actionName, parameters, createdBefore, runAt)
	if err != nil {
		return err
	}

	return c.execution.CreateRequest(c.tx, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
			Parameters: parameters,
		},
	}, &runAt)
}
