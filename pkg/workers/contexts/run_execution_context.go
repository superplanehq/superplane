package contexts

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type RunExecutionContext struct {
	tx                    *gorm.DB
	canvas                *models.Canvas
	node                  *models.CanvasNode
	execution             *models.CanvasNodeExecution
	maxCrossWorkflowDepth int
	onPendingRunCreated   func(workflowID, runID uuid.UUID)
	onRunCancelled        func(workflowID, runID uuid.UUID, drainResult *models.RunCancellationDrainResult)
}

func NewRunExecutionContext(tx *gorm.DB, canvas *models.Canvas, node *models.CanvasNode, execution *models.CanvasNodeExecution) *RunExecutionContext {
	return &RunExecutionContext{
		tx:                    tx,
		canvas:                canvas,
		node:                  node,
		execution:             execution,
		maxCrossWorkflowDepth: 8,
	}
}

func (c *RunExecutionContext) WithPendingRunCreated(fn func(workflowID, runID uuid.UUID)) *RunExecutionContext {
	c.onPendingRunCreated = fn
	return c
}

func (c *RunExecutionContext) WithRunCancelled(fn func(workflowID, runID uuid.UUID, drainResult *models.RunCancellationDrainResult)) *RunExecutionContext {
	c.onRunCancelled = fn
	return c
}

func (c *RunExecutionContext) Cancel() error {
	childRuns, err := models.ListChildRunsByParentExecutions(
		c.tx,
		c.canvas.ID,
		[]uuid.UUID{c.execution.ID},
	)
	if err != nil {
		return fmt.Errorf("list child runs: %w", err)
	}

	for _, childRun := range childRuns {
		run, err := models.LockCanvasRunInTransaction(c.tx, childRun.ID)
		if err != nil {
			return fmt.Errorf("lock child run %s: %w", childRun.ID, err)
		}

		if run.State == models.CanvasRunStateFinished {
			continue
		}

		drainResult, err := run.DrainForCancellation(c.tx, c.execution.CancelledBy)
		if err != nil {
			return fmt.Errorf("cancel child run %s: %w", childRun.ID, err)
		}

		if err := run.MarkAsCancelling(c.tx, c.execution.CancelledBy); err != nil {
			return fmt.Errorf("mark child run %s cancelling: %w", childRun.ID, err)
		}

		if c.onRunCancelled != nil {
			c.onRunCancelled(run.WorkflowID, run.ID, drainResult)
		}
	}

	return nil
}

func (c *RunExecutionContext) Create(params core.RunCreationParams) (*core.Run, error) {
	appContext := NewAppContext(c.tx, c.canvas, c.node)

	app, err := appContext.Get(params.App)
	if err != nil {
		return nil, err
	}

	version, err := models.FindLiveCanvasVersionInTransaction(c.tx, uuid.MustParse(app.ID))
	if err != nil {
		return nil, err
	}

	node, err := appContext.GetNode(params.App, params.Node)
	if err != nil {
		return nil, err
	}

	targetWorkflowID := uuid.MustParse(app.ID)
	err = models.ValidateSubRunCreation(
		c.tx,
		c.execution.RunID,
		targetWorkflowID,
		node.ID,
		c.maxCrossWorkflowDepth,
	)
	if err != nil {
		return nil, err
	}

	run := &models.CanvasRun{
		ID:                uuid.New(),
		WorkflowID:        targetWorkflowID,
		NodeID:            node.ID,
		VersionID:         version.ID,
		ParentRunID:       &c.execution.RunID,
		ParentWorkflowID:  &c.canvas.ID,
		ParentExecutionID: &c.execution.ID,
		Callbacks:         params.Callbacks,
		Input:             models.NewJSONValue(params.Input),
		State:             models.CanvasRunStatePending,
	}

	err = c.tx.Create(run).Error
	if err != nil {
		return nil, err
	}

	if c.onPendingRunCreated != nil {
		c.onPendingRunCreated(run.WorkflowID, run.ID)
	}

	return &core.Run{
		ID: run.ID,
	}, nil
}
