package contexts

import (
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
	err = models.ValidateSubRunCreationInTransaction(
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
