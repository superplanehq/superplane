package changesets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// RerunNodeSetup runs Setup again on live nodes. Use it when the node
// configuration is unchanged but a value that Setup resolves has changed,
// for example the repositories of a factory workspace.
func RerunNodeSetup(
	ctx context.Context,
	tx *gorm.DB,
	canvas *models.Canvas,
	nodeIDs []string,
	options CanvasPublisherOptions,
) error {
	if err := options.Validate(); err != nil {
		return err
	}
	if len(nodeIDs) == 0 {
		return nil
	}

	nodes, err := models.FindCanvasNodesByIDs(tx, canvas.ID, nodeIDs)
	if err != nil {
		return err
	}

	publisher := &CanvasPublisher{tx: tx, options: options, canvas: canvas}
	for i := range nodes {
		node := &nodes[i]
		if err := publisher.setupNode(ctx, node); err != nil {
			message := err.Error()
			node.State = models.CanvasNodeStateError
			node.StateReason = &message
		} else if node.State == models.CanvasNodeStateError {
			node.State = models.CanvasNodeStateReady
			node.StateReason = nil
		}
		if err := tx.Save(node).Error; err != nil {
			return err
		}
	}
	return nil
}
