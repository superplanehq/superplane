package contexts

import (
	"encoding/json"
	"errors"
	"fmt"
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

	// Optional websocket fan-out callback: invoked after every successful
	// mutation with a `reason` string (currently the event type that was
	// recorded). Wired by the node executor via WithWorkOrderUpdated.
	onWorkOrderUpdated func(factoryID, orderID, reason string)

	lineStepOnce   bool
	lineStepLoaded bool
	lineStepCache  lineStepInfo
}

type lineStepInfo struct {
	LineID    uuid.UUID
	LineName  string
	StepIndex int
	StepName  string
}

func NewFactoryContext(tx *gorm.DB, canvas *models.Canvas, execution *models.CanvasNodeExecution) *FactoryContext {
	return &FactoryContext{
		tx:        tx,
		canvas:    canvas,
		execution: execution,
	}
}

func (c *FactoryContext) WithWorkOrderUpdated(callback func(factoryID, orderID, reason string)) *FactoryContext {
	c.onWorkOrderUpdated = callback
	return c
}

func (c *FactoryContext) CreateWorkOrder(params core.WorkOrderParams) (*core.WorkOrder, error) {
	// A run already tied to another work order must not spawn a new one.
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

	sourceRunID := c.execution.RunID
	order, err := f.CreateWorkOrder(c.tx, params.Title, params.Description, nil, []uuid.UUID{}, &sourceRunID)
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(f.ID, order.ID, factory.EventTypeOrderStatusUpdated)
	return workOrderToCore(order), nil
}

func (c *FactoryContext) UpdateWorkOrderStatus(params core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, false, err
	}

	changed, err := order.UpdateStatus(c.tx, models.FactoryWorkOrderStatusUpdate{
		ToState:    params.State,
		Result:     params.Result,
		Automation: c.automationRef(),
		Run:        c.runRef(),
		App:        c.appRef(),
		SkipSame:   true,
	})
	if err != nil {
		return nil, false, err
	}

	// A SkipSame no-op leaves the row untouched — skipping the fan-out
	// keeps downstream nodes from re-firing on a phantom transition when
	// the component is re-run.
	if !changed {
		return workOrderToCore(order), false, nil
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderStatusUpdated)
	return workOrderToCore(order), true, nil
}

func (c *FactoryContext) AddWorkOrderComment(params core.AddWorkOrderCommentParams) error {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(params.Body)
	if body == "" {
		return errors.New("comment body is required")
	}

	// Canvas comments always attribute to `automation`; user comments
	// only come from the interactive API path.
	author := factory.WorkOrderCommentAuthor{
		Kind:       factory.CommentAuthorKindAutomation,
		Automation: c.automationRef(),
	}

	if err := order.RecordCommentAdded(c.tx, body, author, c.runRef()); err != nil {
		return err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderCommentAdded)
	return nil
}

func (c *FactoryContext) AddWorkOrderArtifact(params core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	artifact, err := order.CreateArtifact(c.tx, models.FactoryWorkOrderArtifactParams{
		Type:       params.Type,
		Data:       params.Data,
		Key:        params.Key,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderArtifactAdded)
	return artifactToCore(artifact)
}

func (c *FactoryContext) UpdateWorkOrderArtifact(params core.UpdateWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	artifact, err := order.UpdateArtifactData(c.tx, params.Key, params.Data)
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderArtifactUpdated)
	return artifactToCore(artifact)
}

func (c *FactoryContext) ReportWorkOrderCheck(params core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	order, err := c.resolveWorkOrder(params.OrderID)
	if err != nil {
		return nil, err
	}

	check, err := order.ReportCheck(c.tx, models.FactoryWorkOrderCheckParams{
		Key:        params.CheckKey,
		Name:       params.Name,
		Score:      params.Score,
		MaxScore:   params.MaxScore,
		Format:     params.Format,
		Level:      params.Level,
		Summary:    params.Summary,
		Analysis:   params.Analysis,
		Automation: c.automationRef(),
		Run:        c.runRef(),
	})
	if err != nil {
		return nil, err
	}

	c.notifyWorkOrderUpdated(order.FactoryID, order.ID, factory.EventTypeOrderCheckReported)
	return checkToCore(check), nil
}

// FindWorkOrder resolves a work order by id or by an artifact key,
// independent of the current run's `factory_work_order_executions` row.
// This is what lets a plain webhook-triggered run (e.g. github.onPullRequest)
// locate a work order to act on.
func (c *FactoryContext) FindWorkOrder(params core.FindWorkOrderParams) (*core.WorkOrder, error) {
	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	var order *models.FactoryWorkOrder
	switch params.By {
	case "id":
		orderID, parseErr := uuid.Parse(params.OrderID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid orderId %q: %w", params.OrderID, parseErr)
		}
		order, err = f.FindWorkOrder(c.tx, orderID)
	case "artifactKey":
		order, err = f.FindWorkOrderByArtifactKey(c.tx, params.ArtifactKey)
	default:
		return nil, fmt.Errorf("unknown findWorkOrder lookup %q", params.By)
	}
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
			return nil, core.ErrWorkOrderNotFound
		}
		return nil, err
	}

	return workOrderToCore(order), nil
}

// resolveWorkOrder resolves the work order a mutation should target.
// orderID is always required and explicit — the component field that
// feeds it defaults to `{{ order().id }}` (the current run's work order)
// but every caller must resolve and pass a real id, which is what lets a
// run not attached to any `factory_work_order_executions` row (e.g. a
// plain github.onPullRequest webhook run) still target a specific order.
func (c *FactoryContext) resolveWorkOrder(orderID string) (*models.FactoryWorkOrder, error) {
	if orderID == "" {
		return nil, errors.New("orderId is required")
	}

	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid orderId %q: %w", orderID, err)
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	order, err := f.FindWorkOrder(c.tx, id)
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderNotFound) {
			return nil, fmt.Errorf("work order %q not found", orderID)
		}
		return nil, err
	}

	return order, nil
}

func (c *FactoryContext) notifyWorkOrderUpdated(factoryID, orderID uuid.UUID, reason string) {
	if c.onWorkOrderUpdated == nil {
		return
	}
	c.onWorkOrderUpdated(factoryID.String(), orderID.String(), reason)
}

// runRef attributes emitted events back to the currently executing run.
func (c *FactoryContext) runRef() *factory.RunRef {
	if c.execution == nil {
		return nil
	}

	return &factory.RunRef{
		ID: c.execution.RunID,
	}
}

// appRef attributes emitted events back to the canvas ("app") the current
// run belongs to. Paired with runRef, this lets the timeline link straight
// to the originating run.
func (c *FactoryContext) appRef() *factory.AppRef {
	if c.canvas == nil {
		return nil
	}

	return &factory.AppRef{ID: c.canvas.ID}
}

// automationRef captures node/app/line/step identity for timeline
// attribution. Missing fields are tolerated so the event still lands.
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

	if info, ok := c.lineStep(); ok {
		stepIndex := info.StepIndex
		ref.LineID = info.LineID
		ref.LineName = info.LineName
		ref.StepIndex = &stepIndex
		ref.StepName = info.StepName
	}

	return ref
}

// lineStep resolves the factory line + step that owns the current run
// so events can be attributed back to the pipeline the user configured.
// Runs not attached to a work order (e.g. `CreateWorkOrder`) return
// ok=false and callers omit the fields.
func (c *FactoryContext) lineStep() (lineStepInfo, bool) {
	if c.lineStepOnce {
		return c.lineStepCache, c.lineStepLoaded
	}
	c.lineStepOnce = true

	if c.execution == nil || c.canvas == nil || c.canvas.FactoryID == nil {
		return lineStepInfo{}, false
	}

	execution, err := models.FindWorkOrderExecutionByRunID(c.tx, c.execution.RunID)
	if err != nil {
		return lineStepInfo{}, false
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return lineStepInfo{}, false
	}

	line, err := f.FindLine(c.tx, execution.LineID)
	if err != nil {
		return lineStepInfo{}, false
	}

	c.lineStepCache = lineStepInfo{
		LineID:    line.ID,
		LineName:  line.Name,
		StepIndex: execution.StepIndex,
		StepName:  execution.StepName,
	}
	c.lineStepLoaded = true
	return c.lineStepCache, true
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
		Data:        data,
	}, nil
}

func checkToCore(check *models.FactoryWorkOrderCheck) *core.WorkOrderCheck {
	return &core.WorkOrderCheck{
		ID:            check.ID.String(),
		WorkOrderID:   check.WorkOrderID.String(),
		Key:           check.Key,
		Name:          check.Name,
		Score:         check.Score,
		MaxScore:      check.MaxScore,
		Format:        check.Format,
		Level:         check.Level,
		PreviousScore: check.PreviousScore,
	}
}
