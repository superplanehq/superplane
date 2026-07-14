package contexts

import (
	"errors"

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

func (c *AppContext) getAppByID(id uuid.UUID) (*core.App, error) {
	otherApp, err := models.FindCanvasInTransaction(c.tx, c.canvas.OrganizationID, id)
	if err == nil {
		return &core.App{
			ID:   otherApp.ID.String(),
			Name: otherApp.Name,
		}, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}

	return nil, err
}

func (c *AppContext) getAppByName(name string) (*core.App, error) {
	otherApp, err := models.FindCanvasByNameInTransaction(c.tx, name, c.canvas.OrganizationID)
	if err == nil {
		return &core.App{
			ID:   otherApp.ID.String(),
			Name: otherApp.Name,
		}, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}

	return nil, err
}

func (c *AppContext) Subscribe(id string) error {
	sourceApp, err := c.Get(id)
	if err != nil {
		return err
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
