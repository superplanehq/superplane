package contexts

import (
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type BackgroundRequestContext struct {
	db        *gorm.DB
	request   *models.CanvasNodeRequest
	node      *models.CanvasNode
	execution *models.CanvasNodeExecution
}

func NewBackgroundRequestContext(db *gorm.DB, request *models.CanvasNodeRequest, node *models.CanvasNode, execution *models.CanvasNodeExecution) *BackgroundRequestContext {
	return &BackgroundRequestContext{db: db, request: request, node: node, execution: execution}
}

func (c *BackgroundRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	if c.execution != nil {
		state := NewBackgroundExecutionStateContext(c.db, c.request, c.execution, nil)
		return state.mutate(func(current *ExecutionStateContext) error {
			return NewExecutionRequestContext(current.tx, current.execution).
				WithCurrentRequest(c.request).
				ScheduleActionCall(actionName, parameters, interval)
		})
	}
	return c.db.Transaction(func(tx *gorm.DB) error {
		if err := lockBackgroundRequestLease(tx, c.request); err != nil {
			return err
		}
		node, err := models.FindUnscopedCanvasNode(tx, c.node.WorkflowID, c.node.NodeID)
		if err != nil {
			return err
		}
		if node.DeletedAt.Valid {
			return c.request.Complete(tx)
		}
		return NewNodeRequestContext(tx, node).ScheduleActionCall(actionName, parameters, interval)
	})
}

func lockBackgroundRequestLease(tx *gorm.DB, request *models.CanvasNodeRequest) error {
	err := models.LockNodeRequestLease(tx, request)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.ErrBackgroundHookLeaseLost
	}
	return err
}
