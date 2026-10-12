package contexts

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
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

	for i := range childRuns {
		run := &childRuns[i]
		cancellation, err := run.RequestCancellation(c.tx, c.execution.CancelledBy)
		if err != nil {
			return fmt.Errorf("cancel child run %s: %w", run.ID, err)
		}

		if c.onRunCancelled != nil && cancellation.Drain != nil {
			c.onRunCancelled(run.WorkflowID, run.ID, cancellation.Drain)
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

func (c *RunExecutionContext) AddError(message string) error {
	if c.execution == nil {
		return fmt.Errorf("add run error: execution is required")
	}

	run, err := models.FindCanvasRunInTransaction(c.tx, c.execution.WorkflowID, c.execution.RunID)
	if err != nil {
		return err
	}

	return run.AddError(c.tx, message, config.MaxPayloadSize())
}
