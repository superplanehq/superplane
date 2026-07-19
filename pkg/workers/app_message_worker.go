package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type AppMessageWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	logger    *log.Entry
}

func NewAppMessageWorker(registry *registry.Registry) *AppMessageWorker {
	return &AppMessageWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "AppMessageWorker"}),
	}
}

func (w *AppMessageWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			appMessages, err := models.ListAppMessages(database.Conn())
			if err != nil {
				w.logger.Errorf("Error listing pending app messages: %v", err)
				continue
			}

			for _, message := range appMessages {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(message models.AppMessage) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessMessage(message); err != nil {
						w.logger.Errorf("Error processing app message %s: %v", message.ID, err)
					}
				}(message)
			}
		}
	}
}

func (w *AppMessageWorker) LockAndProcessMessage(message models.AppMessage) error {
	logger := w.logger.WithFields(log.Fields{"app_message": message.ID})
	logger.Infof("Locking and processing app message")

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockAppMessage(tx, message.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("App message already processed - skipping")
				return nil
			}

			return err
		}

		canvas, err := models.FindUnscopedCanvasInTransaction(tx, locked.CanvasID)
		if err != nil {
			return fmt.Errorf("find source canvas: %w", err)
		}

		node, err := models.FindUnscopedCanvasNode(tx, locked.CanvasID, locked.NodeID)
		if err != nil {
			return fmt.Errorf("find source node: %w", err)
		}

		if node.DeletedAt.Valid {
			logger.Infof("Source node %s deleted - deleting app message", locked.NodeID)
			return locked.Delete(tx)
		}

		var payload any
		if err := json.Unmarshal(locked.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}

		if err := w.deliverBroadcast(tx, canvas, node, locked, payload, onNewEvents); err != nil {
			return err
		}

		return locked.Delete(tx)
	})
	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return nil
}

func (w *AppMessageWorker) deliverBroadcast(tx *gorm.DB, canvas *models.Canvas, sourceNode *models.CanvasNode, appMessage *models.AppMessage, payload any, onNewEvents func([]models.CanvasEvent)) error {
	subs := []models.CanvasSubscription{}
	err := tx.
		Where("source_canvas_id = ?", appMessage.CanvasID).
		Find(&subs).
		Error
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		return nil
	}

	nodesByKey, err := w.findTargetNodes(tx, subs)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		node, ok := nodesByKey[nodeKey{canvasID: sub.TargetCanvasID, nodeID: sub.TargetNodeID}]
		if !ok {
			w.logger.Warnf(
				"skipping broadcast subscription with missing or deleted target node %s on canvas %s",
				sub.TargetNodeID,
				sub.TargetCanvasID,
			)

			if err := models.DeleteCanvasSubscriptionsForNode(tx, sub.TargetCanvasID, sub.TargetNodeID); err != nil {
				w.logger.Errorf("delete stale canvas subscription: %v", err)
			}

			continue
		}

		ref := node.Ref.Data()
		if ref.Trigger == nil || ref.Trigger.Name != "onBroadcast" {
			continue
		}

		if node.State == models.CanvasNodeStateError {
			continue
		}

		if err := w.sendMessageToNode(tx, canvas, sourceNode, node, payload, onNewEvents); err != nil {
			w.logger.Errorf("send broadcast to node: %v", err)
		}
	}

	return nil
}

type nodeKey struct {
	canvasID uuid.UUID
	nodeID   string
}

func (w *AppMessageWorker) findTargetNodes(tx *gorm.DB, subs []models.CanvasSubscription) (map[nodeKey]*models.CanvasNode, error) {
	nodeIDsByCanvas := map[uuid.UUID][]string{}
	for _, sub := range subs {
		nodeIDsByCanvas[sub.TargetCanvasID] = append(nodeIDsByCanvas[sub.TargetCanvasID], sub.TargetNodeID)
	}

	nodesByKey := make(map[nodeKey]*models.CanvasNode, len(subs))
	for canvasID, nodeIDs := range nodeIDsByCanvas {
		nodes, err := models.FindCanvasNodesByIDs(tx, canvasID, uniqueStrings(nodeIDs))
		if err != nil {
			return nil, err
		}

		for i := range nodes {
			key := nodeKey{canvasID: canvasID, nodeID: nodes[i].NodeID}
			nodesByKey[key] = &nodes[i]
		}
	}

	return nodesByKey, nil
}

func (w *AppMessageWorker) sendMessageToNode(tx *gorm.DB, sourceCanvas *models.Canvas, sourceNode *models.CanvasNode, targetNode *models.CanvasNode, payload any, onNewEvents func([]models.CanvasEvent)) error {
	ref := targetNode.Ref.Data()
	if targetNode.Type != models.NodeTypeTrigger || ref.Trigger == nil {
		return nil
	}

	triggerName := ref.Trigger.Name
	trigger, err := w.registry.GetTrigger(triggerName)
	if err != nil {
		return fmt.Errorf("trigger %s not found", triggerName)
	}

	message := map[string]any{
		"payload": payload,
		"app": map[string]any{
			"id":   sourceCanvas.ID.String(),
			"name": sourceCanvas.Name,
		},
		"node": map[string]any{
			"id":   sourceNode.NodeID,
			"name": sourceNode.Name,
		},
	}

	appTrigger, ok := trigger.(core.AppTrigger)
	if !ok {
		return nil
	}

	return appTrigger.OnAppMessage(core.AppMessageContext{
		HTTP:          w.registry.HTTPContextInTransaction(tx),
		Configuration: targetNode.Configuration.Data(),
		NodeMetadata:  contexts.NewNodeMetadataContext(tx, targetNode),
		Message:       message,
		Events:        contexts.NewEventContext(tx, targetNode, nil, onNewEvents),
		Logger:        logging.ForNode(*targetNode),
	})
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		unique = append(unique, value)
	}

	return unique
}
