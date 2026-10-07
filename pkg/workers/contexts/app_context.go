package contexts

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type AppContext struct {
	tx     *gorm.DB
	canvas *models.Canvas
	node   *models.CanvasNode
}

func NewAppContext(tx *gorm.DB, canvas *models.Canvas, node *models.CanvasNode) *AppContext {
	return &AppContext{
		tx:     tx,
		canvas: canvas,
		node:   node,
	}
}

func (c *AppContext) Get(idOrName string) (*core.App, error) {
	id, err := uuid.Parse(idOrName)
	if err == nil {
		return c.getAppByID(id)
	}

	return c.getAppByName(idOrName)
}

func (c *AppContext) GetNode(app, node string) (*core.CanvasNode, error) {
	other, err := c.Get(app)
	if err != nil {
		return nil, err
	}

	otherNode, err := models.FindCanvasNode(c.tx, uuid.MustParse(other.ID), node)
	if err != nil {
		return nil, err
	}

	return &core.CanvasNode{
		ID:   otherNode.NodeID,
		Name: otherNode.Name,
	}, nil
}

func (c *AppContext) getAppByID(id uuid.UUID) (*core.App, error) {
	otherApp, err := models.FindCanvasInTransaction(c.tx, c.canvas.OrganizationID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, core.ErrNotFound
		}

		return nil, err
	}

	if err := c.validateFactoryOwnedAppReference(c.canvas, otherApp); err != nil {
		return nil, err
	}

	return &core.App{
		ID:   otherApp.ID.String(),
		Name: otherApp.Name,
	}, nil
}

func (c *AppContext) getAppByName(name string) (*core.App, error) {
	otherApp, err := models.FindCanvasByName(c.tx, c.canvas.OrganizationID, c.canvas.FactoryID, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, core.ErrNotFound
		}

		return nil, err
	}

	if err := c.validateFactoryOwnedAppReference(c.canvas, otherApp); err != nil {
		return nil, err
	}

	return &core.App{
		ID:   otherApp.ID.String(),
		Name: otherApp.Name,
	}, nil
}

func (c *AppContext) validateFactoryOwnedAppReference(caller, target *models.Canvas) error {
	if caller.FactoryID == nil {
		if target.FactoryID != nil {
			return fmt.Errorf("app %q is owned by a factory", target.Name)
		}

		return nil
	}

	if target.FactoryID == nil || *target.FactoryID != *caller.FactoryID {
		return fmt.Errorf("app %q is not owned by this factory", target.Name)
	}

	return nil
}

func (c *AppContext) Subscribe(id string) error {
	sourceApp, err := c.Get(id)
	if err != nil {
		return err
	}

	if sourceApp.ID == c.canvas.ID.String() {
		return fmt.Errorf("cannot self-subscribe to messages")
	}

	err = models.DeleteCanvasSubscriptionsForNode(c.tx, c.canvas.ID, c.node.NodeID)
	if err != nil {
		return err
	}

	sub := &models.CanvasSubscription{
		SourceCanvasID: uuid.MustParse(sourceApp.ID),
		TargetCanvasID: c.canvas.ID,
		TargetNodeID:   c.node.NodeID,
	}

	return c.tx.Create(sub).Error
}

func (c *AppContext) Unsubscribe() error {
	return models.DeleteCanvasSubscriptionsForNode(c.tx, c.canvas.ID, c.node.NodeID)
}
