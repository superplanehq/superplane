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
	order, err := c.findWorkOrder(params.WorkOrderID)
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
	order, err := c.findWorkOrder(params.WorkOrderID)
	if err != nil {
		return err
	}

	body := strings.TrimSpace(params.Body)
	if body == "" {
		return errors.New("comment body is required")
	}

	kind := params.AuthorKind
	if kind == "" {
		kind = factory.CommentAuthorKindLLM
	}
	if !models.IsValidWorkOrderCommentAuthorKind(kind) {
		return fmt.Errorf("invalid comment author kind %q", kind)
	}
	//
	// The canvas has no acting human, so it must never attribute a comment
	// to `user` — the timeline would render "Someone" and downstream LLM
	// context would attribute the note to a phantom person. Human
	// attribution is only available via the interactive API path, which
	// pulls the id from the authenticated caller.
	//
	if kind == factory.CommentAuthorKindUser {
		return fmt.Errorf("canvas comments cannot be attributed to a human user; use `llm` or `system`")
	}

	author := factory.WorkOrderCommentAuthor{
		Kind:  kind,
		Label: strings.TrimSpace(params.AuthorLabel),
	}

	return order.RecordCommentAdded(c.tx, body, author, c.runRef())
}

func (c *FactoryContext) AddWorkOrderArtifact(params core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	order, err := c.findWorkOrder(params.WorkOrderID)
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

func (c *FactoryContext) findWorkOrder(workOrderID string) (*models.FactoryWorkOrder, error) {
	if c.canvas.FactoryID == nil {
		return nil, errors.New("app is not owned by a factory")
	}

	id, err := uuid.Parse(strings.TrimSpace(workOrderID))
	if err != nil {
		return nil, fmt.Errorf("invalid work order id: %w", err)
	}

	f, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	return f.FindWorkOrder(c.tx, id)
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
