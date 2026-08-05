package contexts

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

type FactoryContext struct {
	tx        *gorm.DB
	canvas    *models.Canvas
	execution *models.CanvasNodeExecution
}

func NewFactoryContext(tx *gorm.DB, canvas *models.Canvas, execution *models.CanvasNodeExecution) *FactoryContext {
	return &FactoryContext{
		tx:        tx,
		canvas:    canvas,
		execution: execution,
	}
}

func (c *FactoryContext) CreateWorkOrder(params core.WorkOrderParams) (*core.WorkOrder, error) {
	//
	// Sanity check: do not allow work order to be created
	// when it is already part of a work order execution
	//
	_, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err == nil {
		return nil, errors.New("cannot create work order while executing another work order")
	}
	if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		return nil, err
	}

	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	order, err := f.CreateWorkOrder(c.tx, params.Title, params.Description, nil, []uuid.UUID{})
	if err != nil {
		return nil, err
	}

	return workOrderToCore(order), nil
}

func (c *FactoryContext) UpdateWorkOrderStatus(params core.UpdateWorkOrderStatusParams) (*core.WorkOrder, error) {
	order, err := c.currentWorkOrder()
	if err != nil {
		return nil, err
	}

	err = order.UpdateStatus(c.tx, models.FactoryWorkOrderStatusUpdate{
		ToState:  params.State,
		Result:   params.Result,
		Run:      c.runRef(),
		SkipSame: true,
	})
	if err != nil {
		return nil, err
	}

	return workOrderToCore(order), nil
}

func (c *FactoryContext) AddWorkOrderComment(params core.AddWorkOrderCommentParams) error {
	order, err := c.currentWorkOrder()
	if err != nil {
		return err
	}

	body := strings.TrimSpace(params.Body)
	if body == "" {
		return errors.New("comment body is required")
	}

	//
	// Canvas comments are always attributed to `automation` — the tool that
	// wrote the note is exposed through the `Automation` payload (node +
	// app), not a free-form `llm` / `system` enum. Human attribution is
	// only available through the interactive API, which reads the
	// authenticated caller's id and writes `kind = user`.
	//
	author := factory.WorkOrderCommentAuthor{
		Kind:       factory.CommentAuthorKindAutomation,
		Automation: c.automationRef(),
	}

	return order.RecordCommentAdded(c.tx, body, author, c.runRef())
}

func (c *FactoryContext) AddWorkOrderArtifact(params core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	order, err := c.currentWorkOrder()
	if err != nil {
		return nil, err
	}

	artifact, err := order.CreateArtifact(c.tx, models.FactoryWorkOrderArtifactParams{
		Type:  params.Type,
		URL:   params.URL,
		Title: params.Title,
		Body:  params.Body,
		Data:  params.Data,
		Run:   c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	return artifactToCore(artifact)
}

// currentWorkOrder resolves the work order that owns the currently
// executing canvas run. Factory work-order components (update status,
// add comment, add artifact) only make sense in the context of a
// running work order — we look up that link through the run rather
// than asking the component author to supply the ID by hand.
//
// The link is the `factory_work_order_executions` row that
// `DispatchWorkOrder` created for this run. Canvas runs that were not
// started as part of a work-order dispatch have no such row; in that
// case we return a clear error so the component fails fast instead of
// silently writing to nothing.
func (c *FactoryContext) currentWorkOrder() (*models.FactoryWorkOrder, error) {
	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}
	if c.execution == nil {
		return nil, errors.New("factory context has no current execution")
	}

	execution, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
			return nil, errors.New("this canvas run is not attached to a work order; dispatch a work order to it first")
		}
		return nil, err
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	return f.FindWorkOrder(c.tx, execution.WorkOrderID)
}

// runRef returns a lightweight reference to the current run so events written
// from a canvas can be attributed back to the emitting execution.
func (c *FactoryContext) runRef() *factory.RunRef {
	if c.execution == nil {
		return nil
	}

	return &factory.RunRef{
		ID: c.execution.RunID,
	}
}

// automationRef captures the identity of the executing canvas node (and
// its owning app) so timeline consumers can render "commented via
// <node> in <app>" without inferring the source from a free-form
// author label. Failure to resolve the node is soft: we still emit the
// comment with whatever we know rather than dropping the whole event.
func (c *FactoryContext) automationRef() *factory.AutomationRef {
	if c.execution == nil || c.canvas == nil {
		return nil
	}

	ref := &factory.AutomationRef{
		NodeID:  c.execution.NodeID,
		AppID:   c.canvas.ID,
		AppName: c.canvas.Name,
	}

	node, err := c.canvas.FindNode(c.execution.NodeID)
	if err == nil && node != nil {
		ref.NodeName = node.Name
	}

	return ref
}

func workOrderToCore(order *models.FactoryWorkOrder) *core.WorkOrder {
	return &core.WorkOrder{
		ID:          order.ID.String(),
		Title:       order.Title,
		Description: order.Description,
		State:       order.State,
		Result:      order.Result,
	}
}

func artifactToCore(artifact *models.FactoryWorkOrderArtifact) (*core.WorkOrderArtifact, error) {
	data := map[string]any{}
	if len(artifact.Data) > 0 {
		if err := json.Unmarshal(artifact.Data, &data); err != nil {
			return nil, err
		}
	}

	return &core.WorkOrderArtifact{
		ID:          artifact.ID.String(),
		WorkOrderID: artifact.WorkOrderID.String(),
		Type:        artifact.Type,
		URL:         artifact.URL,
		Title:       artifact.Title,
		Body:        artifact.Body,
		Data:        data,
	}, nil
}
